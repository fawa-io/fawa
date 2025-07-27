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
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/fawa-io/fawa/newcanva/pkg/fwlog"
	"github.com/fawa-io/fawa/newcanva/pkg/util"
	"github.com/gorilla/websocket"
	"github.com/quic-go/webtransport-go"
)

const (
	sessionCleanerInterval = 1 * time.Minute
	sessionExpiryDuration  = 10 * time.Minute
)

// CanvasSession represents a collaborative drawing session
// All clients (WebSocket or WebTransport) join a session by code
// Each session maintains its own clients, history, and broadcast channel

type CanvasSession struct {
	Code       string
	Clients    map[string]*SessionClient
	ClientsMu  sync.RWMutex
	History    []*DrawEvent
	HistoryMu  sync.RWMutex
	Broadcast  chan *DrawEvent
	LastActive time.Time
}

type SessionClient struct {
	ID           string
	ConnType     string // "websocket" or "webtransport"
	WSConn       *websocket.Conn
	WTSession    *webtransport.Session
	OutputStream io.Writer // For WT: *webtransport.Stream, for WS: *websocket.Conn
}

// CanvasServiceHandler manages all canvas sessions
// All connections (WebSocket/WebTransport) join a session by code

type CanvasServiceHandler struct {
	Sessions   map[string]*CanvasSession
	SessionsMu sync.RWMutex
	Upgrader   websocket.Upgrader
	WTServer   *webtransport.Server
}

func NewCanvasServiceHandler() *CanvasServiceHandler {
	h := &CanvasServiceHandler{
		Sessions: make(map[string]*CanvasSession),
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		WTServer: &webtransport.Server{},
	}
	go h.sessionCleaner()
	return h
}

// CreateCanvas creates a new canvas session and returns its code
func (h *CanvasServiceHandler) CreateCanvas(w http.ResponseWriter, r *http.Request) {
	code := util.GenerateRandomString(6)
	session := &CanvasSession{
		Code:       code,
		Clients:    make(map[string]*SessionClient),
		Broadcast:  make(chan *DrawEvent, 100),
		LastActive: time.Now(),
	}
	h.SessionsMu.Lock()
	h.Sessions[code] = session
	h.SessionsMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if _, err := fmt.Fprintf(w, `{"code":"%s"}`, code); err != nil {
		fwlog.Warnf("write response failed: %v", err)
	}
}

// JoinCanvas checks if a session exists for the given code
func (h *CanvasServiceHandler) JoinCanvas(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	h.SessionsMu.RLock()
	_, ok := h.Sessions[code]
	h.SessionsMu.RUnlock()
	if !ok {
		http.Error(w, "Canvas not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleWebSocket handles WebSocket connections, joining a session by code
func (h *CanvasServiceHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing canvas code", http.StatusBadRequest)
		return
	}
	h.SessionsMu.RLock()
	session, ok := h.Sessions[code]
	h.SessionsMu.RUnlock()
	if !ok {
		http.Error(w, "Canvas not found", http.StatusNotFound)
		return
	}
	conn, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		fwlog.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer func() { _ = conn.Close() }()
	clientID := util.GenerateRandomString(8)
	client := &SessionClient{
		ID:       clientID,
		ConnType: "websocket",
		WSConn:   conn,
	}
	session.ClientsMu.Lock()
	session.Clients[clientID] = client
	session.ClientsMu.Unlock()
	defer func() {
		session.ClientsMu.Lock()
		delete(session.Clients, clientID)
		session.ClientsMu.Unlock()
		if err := conn.Close(); err != nil {
			fwlog.Warnf("wsConn close failed: %v", err)
		}
	}()

	// Send initial history
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
		if err := conn.WriteJSON(resp); err != nil {
			fwlog.Warnf("Failed to send initial history: %v", err)
		}
	}

	go h.sessionBroadcastWriter(session, client)
	h.sessionWebSocketReader(session, client)
}

// HandleWebTransport handles WebTransport connections, joining a session by code
func (h *CanvasServiceHandler) HandleWebTransport(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing canvas code", http.StatusBadRequest)
		return
	}
	h.SessionsMu.RLock()
	session, ok := h.Sessions[code]
	h.SessionsMu.RUnlock()
	if !ok {
		http.Error(w, "Canvas not found", http.StatusNotFound)
		return
	}
	wtSession, err := h.WTServer.Upgrade(w, r)
	if err != nil {
		fwlog.Errorf("WebTransport upgrade failed: %v", err)
		http.Error(w, "WebTransport upgrade failed", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := wtSession.CloseWithError(0, "server closed"); err != nil {
			fwlog.Warnf("webSession.CloseWithError failed: %v", err)
		}
	}()
	clientID := util.GenerateRandomString(8)
	// Open a single output stream for this client
	outputStream, err := wtSession.OpenStream()
	if err != nil {
		fwlog.Errorf("Failed to open output stream: %v", err)
		return
	}
	defer func() {
		if err := outputStream.Close(); err != nil {
			fwlog.Warnf("outputStream close failed: %v", err)
		}
	}()
	client := &SessionClient{
		ID:           clientID,
		ConnType:     "webtransport",
		WTSession:    wtSession,
		OutputStream: outputStream,
	}
	session.ClientsMu.Lock()
	session.Clients[clientID] = client
	session.ClientsMu.Unlock()
	defer func() {
		session.ClientsMu.Lock()
		delete(session.Clients, clientID)
		session.ClientsMu.Unlock()
	}()

	// Send initial history
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
		data, err := json.Marshal(resp)
		if err == nil {
			if _, err := outputStream.Write(data); err != nil {
				fwlog.Warnf("Failed to send initial history: %v", err)
			}
		}
	}

	go h.sessionBroadcastWriter(session, client)
	h.sessionWebTransportReader(session, client, r.Context())
}

// sessionBroadcastWriter writes all broadcast events to the client's output stream
func (h *CanvasServiceHandler) sessionBroadcastWriter(session *CanvasSession, client *SessionClient) {
	for event := range session.Broadcast {
		resp := &ClientDrawResponse{DrawEvent: event}
		switch client.ConnType {
		case "websocket":
			if err := client.WSConn.WriteJSON(resp); err != nil {
				fwlog.Warnf("WebSocket WriteJSON failed: %v", err)
				return
			}
		case "webtransport":
			data, err := json.Marshal(resp)
			if err != nil {
				fwlog.Warnf("Marshal failed: %v", err)
				return
			}
			if _, err := client.OutputStream.Write(data); err != nil {
				fwlog.Warnf("WebTransport Write failed: %v", err)
				return
			}
		}
	}
}

// sessionWebSocketReader reads messages from a WebSocket client and broadcasts draw events
func (h *CanvasServiceHandler) sessionWebSocketReader(session *CanvasSession, client *SessionClient) {
	for {
		var request ClientDrawRequest
		if err := client.WSConn.ReadJSON(&request); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			fwlog.Warnf("WebSocket decode error: %v", err)
			return
		}
		if request.DrawEvent != nil {
			h.processSessionDrawEvent(session, client.ID, request.DrawEvent)
		}
	}
}

// sessionWebTransportReader reads messages from a WebTransport client and broadcasts draw events
func (h *CanvasServiceHandler) sessionWebTransportReader(session *CanvasSession, client *SessionClient, ctx context.Context) {
	for {
		stream, err := client.WTSession.AcceptStream(ctx)
		if err != nil {
			return
		}
		dec := json.NewDecoder(stream)
		for {
			var request ClientDrawRequest
			if err := dec.Decode(&request); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				fwlog.Warnf("WebTransport decode error: %v", err)
				return
			}
			if request.DrawEvent != nil {
				h.processSessionDrawEvent(session, client.ID, request.DrawEvent)
			}
		}
	}
}

// processSessionDrawEvent processes a draw event and broadcasts it to all clients in the session
func (h *CanvasServiceHandler) processSessionDrawEvent(session *CanvasSession, clientID string, event *DrawEvent) {
	event.ClientID = clientID
	session.HistoryMu.Lock()
	session.History = append(session.History, event)
	session.HistoryMu.Unlock()
	session.Broadcast <- event
	session.LastActive = time.Now()
}

// sessionCleaner removes expired sessions
func (h *CanvasServiceHandler) sessionCleaner() {
	ticker := time.NewTicker(sessionCleanerInterval)
	defer ticker.Stop()
	for {
		<-ticker.C
		now := time.Now()
		h.SessionsMu.Lock()
		for code, session := range h.Sessions {
			session.ClientsMu.RLock()
			clientCount := len(session.Clients)
			session.ClientsMu.RUnlock()
			if clientCount == 0 && now.Sub(session.LastActive) > sessionExpiryDuration {
				delete(h.Sessions, code)
				fwlog.Infof("Canvas session %s expired and removed", code)
			}
		}
		h.SessionsMu.Unlock()
	}
}
