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
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fawa-io/fawa/newcanva/pkg/config"
	"github.com/fawa-io/fawa/newcanva/pkg/cors"
	"github.com/fawa-io/fawa/newcanva/pkg/fwlog"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

func main() {
	if err := config.InitConfig(); err != nil {
		fwlog.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Get configuration
	cfg := config.Get()

	logLevel, err := fwlog.ParseLevel(cfg.LogLevel)
	if err != nil {
		fwlog.Warnf("Invalid initial log level '%s': %v. Using default.", cfg.LogLevel, err)
	}
	fwlog.SetLevel(logLevel)
	fwlog.Infof("Logger initialized with level: %s", cfg.LogLevel)

	// Load TLS certificate
	tlsConfig := &tls.Config{}
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		fwlog.Fatalf("Failed to load TLS certificate: %v", err)
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	// Create canvas service handler
	canvaHandler := NewCanvasServiceHandler()

	// Create HTTP/3 server for WebTransport
	h3Server := &http3.Server{
		Addr:      cfg.Addr,
		TLSConfig: tlsConfig,
	}

	// Create WebTransport server
	wtServer := &webtransport.Server{
		H3: *h3Server, // H3: *h3Server,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	// Set the WebTransport server in the handler
	canvaHandler.WTServer = wtServer

	// Create HTTP server with CORS support (for WebSocket fallback)
	mux := http.NewServeMux()

	// WebTransport endpoint
	mux.HandleFunc("/webtransport/canva", canvaHandler.HandleWebTransport)

	// WebSocket fallback endpoint
	mux.HandleFunc("/ws/canva", canvaHandler.HandleWebSocket)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"status":"ok","service":"newcanva"}`)); err != nil {
			fwlog.Warnf("write response failed: %v", err)
		}
	})

	mux.HandleFunc("/create", canvaHandler.CreateCanvas)
	mux.HandleFunc("/join", canvaHandler.JoinCanvas)

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	// Create HTTP server with CORS middleware (for WebSocket fallback)
	httpServer := &http.Server{
		Addr:    cfg.Addr,
		Handler: cors.NewCORS().Handler(mux),
	}

	// Setup graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		fwlog.Info("Shutting down server...")

		// Set timeout for server shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Shutdown HTTP/3 server
		if err := h3Server.Close(); err != nil {
			fwlog.Errorf("HTTP/3 server shutdown error: %v", err)
		}

		// Shutdown HTTP server
		if err := httpServer.Shutdown(ctx); err != nil {
			fwlog.Errorf("HTTP server shutdown error: %v", err)
		}

		fwlog.Info("Server shutdown complete")
		os.Exit(0)
	}()

	fwlog.Infof("NewCanva WebTransport server starting on %v", cfg.Addr)
	fwlog.Infof("WebTransport endpoint: https://%s/webtransport/canva", cfg.Addr)
	fwlog.Infof("WebSocket fallback endpoint: wss://%s/ws/canva", cfg.Addr)

	// Start the HTTP/3 server for WebTransport
	go func() {
		if err := h3Server.ListenAndServe(); err != nil && err.Error() != "server closed" {
			fwlog.Errorf("HTTP/3 server error: %v", err)
		}
	}()

	// Start the HTTPS server for WebSocket fallback
	if err := httpServer.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fwlog.Fatalf("Failed to start HTTP server: %v", err)
	}
	go func() {
		log.Println(http.ListenAndServe("localhost:8081", nil))
	}()
}
