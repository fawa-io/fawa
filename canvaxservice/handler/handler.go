// Copyright 2025 The fawa Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"context"
	"errors"
	"io"
	"sync"

	"connectrpc.com/connect"
	"github.com/fawa-io/fwpkg/fwlog"
	"github.com/fawa-io/fwpkg/util"

	canvav1 "github.com/fawa-io/fawa/services/canvaxservice/gen/canva/v1"
)

// CanvaServiceHandler handles canvas service requests
// It manages multiple client connections and drawing history
type CanvaServiceHandler struct {
	// Client connection management
	clients   map[string]*client
	clientsMu sync.RWMutex

	// Drawing history
	history   []*canvav1.DrawEvent
	historyMu sync.RWMutex

	// Channel for broadcasting messages
	broadcast chan *canvav1.DrawEvent
	// Channel for service shutdown
	done chan struct{}
}

type client struct {
	id     string
	stream *connect.BidiStream[canvav1.ClientDrawRequest, canvav1.ClientDrawResponse]
}

// NewCanvaServiceHandler creates a new canvas service handler
func NewCanvaServiceHandler() *CanvaServiceHandler {
	h := &CanvaServiceHandler{
		clients:   make(map[string]*client),
		history:   make([]*canvav1.DrawEvent, 0, 100),
		broadcast: make(chan *canvav1.DrawEvent, 100),
		done:      make(chan struct{}),
	}

	// Start broadcast handling goroutine
	go h.handleBroadcasts()

	return h
}

// Collaborate handles bidirectional streaming for canvas collaboration
// This is the interface method generated from the proto file
func (h *CanvaServiceHandler) Collaborate(
	ctx context.Context,
	stream *connect.BidiStream[canvav1.ClientDrawRequest, canvav1.ClientDrawResponse],
) error {
	// Generate unique client identifier
	clientID := util.Generaterandomstring(8)
	fwlog.Infof("New canvas connection: client %s", clientID)

	// Register client
	h.registerClient(clientID, stream)
	defer h.unregisterClient(clientID)

	fwlog.Debugf("Client %s: Sending initial history", clientID)
	// Send initial history
	if err := h.sendInitialHistory(stream); err != nil {
		fwlog.Errorf("Failed to send history to client %s: %v", clientID, err)
		return err
	}

	fwlog.Debugf("Client %s: Entering message processing loop", clientID)
	// Process client messages
	for {
		// Check if context is canceled
		if err := ctx.Err(); err != nil {
			fwlog.Infof("Client %s context canceled: %v", clientID, err)
			return err
		}

		fwlog.Debugf("Client %s: Waiting to receive message", clientID)
		// Receive client message
		msg, err := stream.Receive()
		if err != nil {
			if errors.Is(err, io.EOF) || connect.CodeOf(err) == connect.CodeCanceled {
				fwlog.Infof("Client %s disconnected", clientID)
				return nil
			}
			fwlog.Errorf("Failed to receive message from client %s: %v", clientID, err)
			return err
		}

		fwlog.Debugf("Client %s: Received message: %+v", clientID, msg)

		// Process drawing events
		if drawEvent := msg.GetDrawEvent(); drawEvent != nil {
			fwlog.Debugf("Client %s: Processing draw event: %+v", clientID, drawEvent)

			// Ensure client ID is set
			drawEvent.ClientId = clientID

			switch drawEvent.Type {
			case "ping":
				fwlog.Debugf("Client %s: Received ping, keeping connection alive", clientID)
			case "clear":
				fwlog.Infof("Client %s: Received clear canvas command", clientID)
				h.addToHistory(drawEvent)
				h.broadcast <- drawEvent
				h.clearHistory(drawEvent)
			default:
				h.addToHistory(drawEvent)
				h.broadcast <- drawEvent
			}
		} else {
			fwlog.Debugf("Client %s: Received non-draw event or empty message", clientID)
		}
	}
}

// Internal helper methods

// Register new client
func (h *CanvaServiceHandler) registerClient(id string, stream *connect.BidiStream[canvav1.ClientDrawRequest, canvav1.ClientDrawResponse]) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	h.clients[id] = &client{
		id:     id,
		stream: stream,
	}
	fwlog.Infof("Client %s registered, active connections: %d", id, len(h.clients))
}

// Unregister client
func (h *CanvaServiceHandler) unregisterClient(id string) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	delete(h.clients, id)
	fwlog.Infof("Client %s unregistered, active connections: %d", id, len(h.clients))
}

// Send initial history
func (h *CanvaServiceHandler) sendInitialHistory(stream *connect.BidiStream[canvav1.ClientDrawRequest, canvav1.ClientDrawResponse]) error {
	h.historyMu.RLock()
	events := make([]*canvav1.DrawEvent, len(h.history))
	copy(events, h.history) // Create copy to avoid holding lock for too long
	h.historyMu.RUnlock()

	history := &canvav1.History{
		Events: events,
	}

	return stream.Send(&canvav1.ClientDrawResponse{
		Message: &canvav1.ClientDrawResponse_InitialHistory{
			InitialHistory: history,
		},
	})
}

// Purge history, but retain the specified purge events
func (h *CanvaServiceHandler) clearHistory(clearEvent *canvav1.DrawEvent) {
	h.historyMu.Lock()
	defer h.historyMu.Unlock()

	//purge All History And Keep Only PurgeEvents
	h.history = []*canvav1.DrawEvent{clearEvent}
	fwlog.Infof("Canvas history cleared by client %s", clearEvent.ClientId)
}

// Add event to history
func (h *CanvaServiceHandler) addToHistory(event *canvav1.DrawEvent) {
	h.historyMu.Lock()
	defer h.historyMu.Unlock()

	// Implement history size limit
	if len(h.history) >= 1000 {
		// If history gets too large, remove old events or implement persistence
		h.history = h.history[len(h.history)/2:]
	}

	h.history = append(h.history, event)
}

// Handle broadcast messages
func (h *CanvaServiceHandler) handleBroadcasts() {
	for {
		select {
		case event := <-h.broadcast:
			h.broadcastToClients(event)
		case <-h.done:
			fwlog.Info("Canvas service broadcast goroutine exiting")
			return
		}
	}
}

// Broadcast event to all clients
func (h *CanvaServiceHandler) broadcastToClients(event *canvav1.DrawEvent) {
	message := &canvav1.ClientDrawResponse{
		Message: &canvav1.ClientDrawResponse_DrawEvent{
			DrawEvent: event,
		},
	}

	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	for id, cl := range h.clients {
		// Use anonymous function to avoid defer in loop
		func(clientID string, cl *client) {
			if err := cl.stream.Send(message); err != nil {
				fwlog.Errorf("Failed to send message to client %s: %v", clientID, err)
				// Note: we don't remove the client here because we hold a read lock
				// Client will be automatically unregistered via Collaborate method's defer
			}
		}(id, cl)
	}
}

// Close shuts down the canvas service
// Call this when stopping the service
func (h *CanvaServiceHandler) Close() {
	close(h.done)

	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	// Close all client connections
	h.clients = make(map[string]*client)
	fwlog.Info("Canvas service shut down")
}
