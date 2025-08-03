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
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fawa-io/fwpkg/cors"
	"github.com/fawa-io/fwpkg/fwlog"

	"github.com/fawa-io/fawa/fileservice/config"
	"github.com/fawa-io/fawa/fileservice/gen/file/v1/filev1connect"
	file "github.com/fawa-io/fawa/fileservice/handler"
)

func main() {
	if err := config.InitConfig(); err != nil {
		fwlog.Fatalf("Failed to initialize configuration: %v", err)
	}

	cfg := config.Get()

	logLevel, err := fwlog.ParseLevel(cfg.LogLevel)
	if err != nil {
		fwlog.Warnf("Invalid initial log level '%s': %v. Using default.", cfg.LogLevel, err)
	}
	fwlog.SetLevel(logLevel)
	fwlog.Infof("Logger initialized with level: %s", cfg.LogLevel)

	fileSvcHdr := &file.FileServiceHandler{}
	fileProcedure, fileHandler := filev1connect.NewFileServiceHandler(fileSvcHdr)

	mux := http.NewServeMux()
	mux.Handle(fileProcedure, fileHandler)

	fileSrv := &http.Server{
		Addr:    cfg.Addr,
		Handler: cors.NewCORS().Handler(mux),
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		fwlog.Info("Shutting down server...")

		// Close file service
		if err := fileSvcHdr.Close(); err != nil {
			fwlog.Errorf("Error closing file service: %v", err)
		}

		// Set timeout for HTTP server shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := fileSrv.Shutdown(ctx); err != nil {
			fwlog.Errorf("Server shutdown error: %v", err)
		}

		fwlog.Info("Server shutdown complete")
		os.Exit(0)
	}()

	fwlog.Infof("Server starting on %v", cfg.Addr)

	// Check if certificate files exist
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		// Check if certificate files actually exist
		if _, err := os.Stat(cfg.CertFile); err == nil {
			if _, err := os.Stat(cfg.KeyFile); err == nil {
				// Start the HTTPS server.
				fwlog.Infof("Starting HTTPS server with certificates: %s, %s", cfg.CertFile, cfg.KeyFile)
				if err := fileSrv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
					fwlog.Fatalf("Failed to start HTTPS server: %v", err)
				}
				return
			}
		}
		fwlog.Warnf("Certificate files not found, falling back to HTTP mode")
	}

	// Start the HTTP server.
	fwlog.Infof("Starting HTTP server")
	if err := fileSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fwlog.Fatalf("Failed to start HTTP server: %v", err)
	}
}
