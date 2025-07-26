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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fawa-io/fawa/newcanva/pkg/fwlog"
	"github.com/fawa-io/fawa/newcanva/pkg/util"
	"github.com/gorilla/websocket"
	"github.com/quic-go/webtransport-go"
)

type CanvasSession struct {
	Code       string
	Clients    map[string]*client
	ClientsMu  sync.RWMutex
	History    []*DrawEvent
	HistoryMu  sync.RWMutex
	Broadcast  chan *DrawEvent
	Done       chan struct{}
	LastActive time.Time
}

// CanvaServiceHandler handles canvas service requests using WebTransport
type CanvaServiceHandler struct {
	// Client connection management
	clients   map[string]*client
	clientsMu sync.RWMutex

	// Drawing history
	history   []*DrawEvent
	historyMu sync.RWMutex

	// Channel for broadcasting messages
	broadcast chan *DrawEvent
	// Channel for service shutdown
	done chan struct{}

	// WebSocket upgrader for fallback
	upgrader websocket.Upgrader

	// WebTransport server
	wtServer *webtransport.Server

	// 画布会话管理
	sessions   map[string]*CanvasSession
	sessionsMu sync.RWMutex
}

type client struct {
	id       string
	session  *WebTransportSession
	stream   *WebTransportStream
	wsConn   *websocket.Conn
	isWS     bool
	sendChan chan []byte
	done     chan struct{}
}

// NewCanvaServiceHandler creates a new canvas service handler
func NewCanvaServiceHandler() *CanvaServiceHandler {
	h := &CanvaServiceHandler{
		clients:   make(map[string]*client),
		history:   make([]*DrawEvent, 0, 100),
		broadcast: make(chan *DrawEvent, 100),
		done:      make(chan struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		wtServer: &webtransport.Server{},
		sessions: make(map[string]*CanvasSession),
	}
	go h.sessionCleaner()
	go h.handleBroadcasts()

	return h
}

// 创建画布
func (h *CanvaServiceHandler) CreateCanvas(w http.ResponseWriter, r *http.Request) {
	code := util.Generaterandomstring(6)
	session := &CanvasSession{
		Code:       code,
		Clients:    make(map[string]*client),
		Broadcast:  make(chan *DrawEvent, 100),
		Done:       make(chan struct{}),
		LastActive: time.Now(),
	}
	h.sessionsMu.Lock()
	h.sessions[code] = session
	h.sessionsMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(fmt.Sprintf(`{"code":"%s"}`, code))); err != nil {
		fwlog.Warnf("write response failed: %v", err)
	}
}

// 加入画布
func (h *CanvaServiceHandler) JoinCanvas(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	h.sessionsMu.RLock()
	_, ok := h.sessions[code]
	h.sessionsMu.RUnlock()
	if !ok {
		http.Error(w, "Canvas not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// WebTransport连接，支持code参数
func (h *CanvaServiceHandler) HandleWebTransport(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing canvas code", http.StatusBadRequest)
		return
	}
	h.sessionsMu.RLock()
	session, ok := h.sessions[code]
	h.sessionsMu.RUnlock()
	if !ok {
		http.Error(w, "Canvas not found", http.StatusNotFound)
		return
	}
	webSession, err := h.wtServer.Upgrade(w, r)
	if err != nil {
		fwlog.Errorf("WebTransport upgrade failed: %v", err)
		http.Error(w, "WebTransport upgrade failed", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := webSession.CloseWithError(0, "server closed"); err != nil {
			fwlog.Warnf("webSession.CloseWithError failed: %v", err)
		}
	}()

	clientID := util.Generaterandomstring(8)
	fwlog.Infof("New WebTransport connection: client %s join canvas %s", clientID, code)

	cl := &client{id: clientID}
	session.ClientsMu.Lock()
	session.Clients[clientID] = cl
	session.ClientsMu.Unlock()
	defer func() {
		session.ClientsMu.Lock()
		delete(session.Clients, clientID)
		session.ClientsMu.Unlock()
	}()

	session.HistoryMu.RLock()
	historyCopy := make([]*DrawEvent, len(session.History))
	copy(historyCopy, session.History)
	session.HistoryMu.RUnlock()
	if len(historyCopy) > 0 {
		resp := &ClientDrawResponse{
			InitialHistory: &History{Events: make([]DrawEvent, len(historyCopy))},
		}
		for i, e := range historyCopy {
			resp.InitialHistory.Events[i] = *e
		}
		data, err := resp.ToJSON()
		if err == nil {
			stream, err := webSession.OpenStream()
			if err == nil {
				if _, err := stream.Write(data); err != nil {
					fwlog.Warnf("stream.Write failed: %v", err)
				}
				if err := stream.Close(); err != nil {
					fwlog.Warnf("stream.Close failed: %v", err)
				}
			}
		}
	}

	ctx := webSession.Context()
	go h.handleSessionSend(ctx, session, clientID, webSession)
	h.handleSessionReceive(ctx, session, clientID, webSession)
}

// HandleWebSocket handles WebSocket connections as fallback
func (h *CanvaServiceHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Generate unique client identifier
	clientID := util.Generaterandomstring(8)
	fwlog.Infof("New WebSocket connection: client %s", clientID)

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fwlog.Errorf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// Create session and stream for WebSocket
	session := &WebTransportSession{
		ID:       clientID,
		SendChan: make(chan []byte, 100),
		Done:     make(chan struct{}),
	}

	stream := &WebTransportStream{
		ID:       clientID,
		ReadChan: make(chan []byte, 100),
		Done:     make(chan struct{}),
	}

	session.Stream = stream

	// Register client
	h.registerClient(clientID, session, stream, conn, true)
	defer h.unregisterClient(clientID)

	fwlog.Debugf("Client %s: Sending initial history", clientID)
	// Send initial history
	if err := h.sendInitialHistory(session); err != nil {
		fwlog.Errorf("Failed to send history to client %s: %v", clientID, err)
		return
	}

	// Start message processing goroutine
	go h.processClientMessages(clientID, session, stream)

	// Handle WebSocket messages
	h.handleWebSocketMessages(clientID, conn, stream)
}

// handleWebSocketMessages handles WebSocket message processing
func (h *CanvaServiceHandler) handleWebSocketMessages(clientID string, conn *websocket.Conn, stream *WebTransportStream) {
	defer conn.Close()

	for {
		// Read message
		_, message, err := conn.ReadMessage()
		if err != nil {
			fwlog.Infof("Client %s disconnected: %v", clientID, err)
			return
		}

		fwlog.Debugf("Client %s: Received message: %s", clientID, string(message))

		// Parse message
		var request ClientDrawRequest
		if err := json.Unmarshal(message, &request); err != nil {
			fwlog.Errorf("Failed to parse message from client %s: %v", clientID, err)
			continue
		}

		// Process drawing events
		if request.DrawEvent != nil {
			fwlog.Debugf("Client %s: Processing draw event: %+v", clientID, request.DrawEvent)

			// Ensure client ID is set
			request.DrawEvent.ClientID = clientID

			switch request.DrawEvent.Type {
			case "ping":
				fwlog.Debugf("Client %s: Received ping, keeping connection alive", clientID)
				// Send pong response
				response := &ClientDrawResponse{
					DrawEvent: &DrawEvent{
						Type:     "pong",
						ClientID: clientID,
						Time:     time.Now().UnixMilli(),
					},
				}
				if responseData, err := response.ToJSON(); err == nil {
					if err := conn.WriteMessage(websocket.TextMessage, responseData); err != nil {
						fwlog.Warnf("conn.WriteMessage failed: %v", err)
					}
				}
			case "clear":
				fwlog.Infof("Client %s: Received clear canvas command", clientID)
				h.addToHistory(request.DrawEvent)
				h.broadcast <- request.DrawEvent
				h.clearHistory(request.DrawEvent)
			default:
				h.addToHistory(request.DrawEvent)
				h.broadcast <- request.DrawEvent
			}
		} else {
			fwlog.Debugf("Client %s: Received non-draw event or empty message", clientID)
		}
	}
}

// processClientMessages processes messages from client streams
func (h *CanvaServiceHandler) processClientMessages(clientID string, session *WebTransportSession, stream *WebTransportStream) {
	for {
		select {
		case message := <-stream.ReadChan:
			fwlog.Debugf("Client %s: Processing stream message: %s", clientID, string(message))

			// Parse message
			var request ClientDrawRequest
			if err := json.Unmarshal(message, &request); err != nil {
				fwlog.Errorf("Failed to parse message from client %s: %v", clientID, err)
				continue
			}

			// Process drawing events
			if request.DrawEvent != nil {
				fwlog.Debugf("Client %s: Processing draw event: %+v", clientID, request.DrawEvent)

				// Ensure client ID is set
				request.DrawEvent.ClientID = clientID

				switch request.DrawEvent.Type {
				case "ping":
					fwlog.Debugf("Client %s: Received ping, keeping connection alive", clientID)
				case "clear":
					fwlog.Infof("Client %s: Received clear canvas command", clientID)
					h.addToHistory(request.DrawEvent)
					h.broadcast <- request.DrawEvent
					h.clearHistory(request.DrawEvent)
				default:
					h.addToHistory(request.DrawEvent)
					h.broadcast <- request.DrawEvent
				}
			} else {
				fwlog.Debugf("Client %s: Received non-draw event or empty message", clientID)
			}

		case <-stream.Done:
			fwlog.Infof("Client %s stream closed", clientID)
			return
		case <-h.done:
			fwlog.Infof("Service shutting down, closing client %s", clientID)
			return
		}
	}
}

// Internal helper methods

// Register new client
func (h *CanvaServiceHandler) registerClient(id string, session *WebTransportSession, stream *WebTransportStream, wsConn *websocket.Conn, isWS bool) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	h.clients[id] = &client{
		id:       id,
		session:  session,
		stream:   stream,
		wsConn:   wsConn,
		isWS:     isWS,
		sendChan: make(chan []byte, 100),
		done:     make(chan struct{}),
	}
	fwlog.Infof("Client %s registered, active connections: %d", id, len(h.clients))
}

// Unregister client
func (h *CanvaServiceHandler) unregisterClient(id string) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	if cl, exists := h.clients[id]; exists {
		close(cl.done)
		if cl.wsConn != nil {
			cl.wsConn.Close()
		}
		delete(h.clients, id)
		fwlog.Infof("Client %s unregistered, active connections: %d", id, len(h.clients))
	}
}

// Send initial history
func (h *CanvaServiceHandler) sendInitialHistory(session *WebTransportSession) error {
	h.historyMu.RLock()
	events := make([]*DrawEvent, len(h.history))
	copy(events, h.history) // Create copy to avoid holding lock for too long
	h.historyMu.RUnlock()

	history := &History{
		Events: make([]DrawEvent, len(events)),
	}

	for i, event := range events {
		history.Events[i] = *event
	}

	response := &ClientDrawResponse{
		InitialHistory: history,
	}

	responseData, err := response.ToJSON()
	if err != nil {
		return err
	}

	// Send to session
	select {
	case session.SendChan <- responseData:
		return nil
	case <-session.Done:
		return fmt.Errorf("session closed")
	}
}

// Clear history, but retain the specified clear events
func (h *CanvaServiceHandler) clearHistory(clearEvent *DrawEvent) {
	h.historyMu.Lock()
	defer h.historyMu.Unlock()

	// Purge all history and keep only clear events
	h.history = []*DrawEvent{clearEvent}
	fwlog.Infof("Canvas history cleared by client %s", clearEvent.ClientID)
}

// Add event to history
func (h *CanvaServiceHandler) addToHistory(event *DrawEvent) {
	h.historyMu.Lock()
	defer h.historyMu.Unlock()

	// Implement history size limit
	if len(h.history) >= 1000 {
		// If history gets too large, remove old events
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
func (h *CanvaServiceHandler) broadcastToClients(event *DrawEvent) {
	message := &ClientDrawResponse{
		DrawEvent: event,
	}

	messageData, err := message.ToJSON()
	if err != nil {
		fwlog.Errorf("Failed to marshal broadcast message: %v", err)
		return
	}

	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	for id, cl := range h.clients {
		// Use anonymous function to avoid defer in loop
		func(clientID string, cl *client) {
			if cl.isWS && cl.wsConn != nil {
				// Send via WebSocket
				if err := cl.wsConn.WriteMessage(websocket.TextMessage, messageData); err != nil {
					fwlog.Errorf("Failed to send message to WebSocket client %s: %v", clientID, err)
				}
			} else {
				// Send via WebTransport session
				select {
				case cl.session.SendChan <- messageData:
					// Message sent successfully
				case <-cl.session.Done:
					fwlog.Debugf("Client %s session closed, skipping broadcast", clientID)
				default:
					fwlog.Errorf("Failed to send message to WebTransport client %s: channel full", clientID)
				}
			}
		}(id, cl)
	}
}

// Close shuts down the canvas service
func (h *CanvaServiceHandler) Close() {
	close(h.done)

	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	// Close all client connections
	for _, cl := range h.clients {
		close(cl.done)
		if cl.wsConn != nil {
			cl.wsConn.Close()
		}
	}
	h.clients = make(map[string]*client)
	fwlog.Info("Canvas service shut down")
}

// 会话消息发送
func (h *CanvaServiceHandler) handleSessionSend(ctx context.Context, session *CanvasSession, clientID string, webSession *webtransport.Session) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-session.Broadcast:
			resp := &ClientDrawResponse{DrawEvent: event}
			data, err := resp.ToJSON()
			if err != nil {
				continue
			}
			stream, err := webSession.OpenStream()
			if err != nil {
				continue
			}
			if _, err := stream.Write(data); err != nil {
				fwlog.Warnf("stream.Write failed: %v", err)
			}
			if err := stream.Close(); err != nil {
				fwlog.Warnf("stream.Close failed: %v", err)
			}
		}
	}
}

// 会话消息接收
func (h *CanvaServiceHandler) handleSessionReceive(ctx context.Context, session *CanvasSession, clientID string, webSession *webtransport.Session) {
	for {
		stream, err := webSession.AcceptStream(ctx)
		if err != nil {
			return
		}
		go func(s *webtransport.Stream) {
			defer func() {
				if err := s.Close(); err != nil {
					fwlog.Warnf("stream.Close failed: %v", err)
				}
			}()
			buf := make([]byte, 4096)
			for {
				n, err := s.Read(buf)
				if err != nil {
					return
				}
				var request ClientDrawRequest
				if err := json.Unmarshal(buf[:n], &request); err != nil {
					continue
				}
				if request.DrawEvent != nil {
					h.processSessionDrawEvent(session, clientID, request.DrawEvent)
				}
			}
		}(stream)
	}
}

// 处理会话内绘图事件
func (h *CanvaServiceHandler) processSessionDrawEvent(session *CanvasSession, clientID string, event *DrawEvent) {
	event.ClientID = clientID
	session.HistoryMu.Lock()
	session.History = append(session.History, event)
	session.HistoryMu.Unlock()
	session.Broadcast <- event
	session.LastActive = time.Now()
}

// 会话过期清理
func (h *CanvaServiceHandler) sessionCleaner() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		<-ticker.C
		now := time.Now()
		h.sessionsMu.Lock()
		for code, session := range h.sessions {
			session.ClientsMu.RLock()
			clientCount := len(session.Clients)
			session.ClientsMu.RUnlock()
			if clientCount == 0 && now.Sub(session.LastActive) > 10*time.Minute {
				close(session.Done)
				delete(h.sessions, code)
				fwlog.Infof("Canvas session %s expired and removed", code)
			}
		}
		h.sessionsMu.Unlock()
	}
}
