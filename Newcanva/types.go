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
	"encoding/json"
	"time"
)

// DrawEvent represents a drawing event
type DrawEvent struct {
	Type     string `json:"type"`
	Color    string `json:"color"`
	Size     int    `json:"size"`
	PrevX    int    `json:"prev_x"`
	PrevY    int    `json:"prev_y"`
	CurrX    int    `json:"curr_x"`
	CurrY    int    `json:"curr_y"`
	ClientID string `json:"client_id"`
	Time     int64  `json:"time"`
}

// History represents the drawing history
type History struct {
	Events []DrawEvent `json:"events"`
}

// ClientDrawRequest represents a client request
type ClientDrawRequest struct {
	DrawEvent *DrawEvent `json:"draw_event,omitempty"`
}

// ClientDrawResponse represents a server response
type ClientDrawResponse struct {
	DrawEvent      *DrawEvent `json:"draw_event,omitempty"`
	InitialHistory *History   `json:"initial_history,omitempty"`
}

// WebTransportSession represents a WebTransport session
type WebTransportSession struct {
	ID       string
	Stream   *WebTransportStream
	SendChan chan []byte
	Done     chan struct{}
}

// WebTransportStream represents a WebTransport stream
type WebTransportStream struct {
	ID       string
	ReadChan chan []byte
	Done     chan struct{}
}

// Message represents a WebTransport message
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// NewDrawEvent creates a new draw event
func NewDrawEvent(eventType, color, clientID string, size, prevX, prevY, currX, currY int) *DrawEvent {
	return &DrawEvent{
		Type:     eventType,
		Color:    color,
		Size:     size,
		PrevX:    prevX,
		PrevY:    prevY,
		CurrX:    currX,
		CurrY:    currY,
		ClientID: clientID,
		Time:     time.Now().UnixMilli(),
	}
}

// ToJSON converts the draw event to JSON
func (e *DrawEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON creates a draw event from JSON
func DrawEventFromJSON(data []byte) (*DrawEvent, error) {
	var event DrawEvent
	err := json.Unmarshal(data, &event)
	return &event, err
}

// ToJSON converts the history to JSON
func (h *History) ToJSON() ([]byte, error) {
	return json.Marshal(h)
}

// FromJSON creates a history from JSON
func HistoryFromJSON(data []byte) (*History, error) {
	var history History
	err := json.Unmarshal(data, &history)
	return &history, err
}

// ToJSON converts the client request to JSON
func (r *ClientDrawRequest) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// FromJSON creates a client request from JSON
func ClientDrawRequestFromJSON(data []byte) (*ClientDrawRequest, error) {
	var request ClientDrawRequest
	err := json.Unmarshal(data, &request)
	return &request, err
}

// ToJSON converts the client response to JSON
func (r *ClientDrawResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// FromJSON creates a client response from JSON
func ClientDrawResponseFromJSON(data []byte) (*ClientDrawResponse, error) {
	var response ClientDrawResponse
	err := json.Unmarshal(data, &response)
	return &response, err
}
