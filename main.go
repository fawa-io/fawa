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

	"github.com/fawa-io/fawa/gen/fawa/canva/v1/canvav1connect"
	"github.com/fawa-io/fawa/gen/fawa/file/v1/filev1connect"
	"github.com/fawa-io/fawa/gen/fawa/greet/v1/greetv1connect"
	"github.com/fawa-io/fawa/pkg/config"
	"github.com/fawa-io/fawa/pkg/cors"
	"github.com/fawa-io/fawa/pkg/fwlog"
	"github.com/fawa-io/fawa/pkg/util"
	"github.com/fawa-io/fawa/service/canva"
	"github.com/fawa-io/fawa/service/file"
	"github.com/fawa-io/fawa/service/greet"
)

func main() {
	if err := config.Initconfig(); err != nil {
		fwlog.Fatalf("Failed to initialize configuration: %v", err)
	}
	// dev mode
	cfg := config.Get()

	logLevel, err := fwlog.ParseLevel(cfg.LogLevel)
	if err != nil {
		fwlog.Warnf("Invalid initial log level '%s': %v. Using default.", cfg.LogLevel, err)
	}
	fwlog.SetLevel(logLevel)
	fwlog.Infof("Logger initialized with level: %s", cfg.LogLevel)

	// Create upload dir for file service.
	if !util.Exist(cfg.UploadDir) {
		if err := util.CreateDir(cfg.UploadDir); err != nil {
			fwlog.Fatal(err)
		}
	}
	fileSvcHdr := &file.FileServiceHandler{
		UploadDir: cfg.UploadDir,
	}
	fileProcedure, fileHandler := filev1connect.NewFileServiceHandler(fileSvcHdr)

	// Greet service (no dependencies yet)
	greetSvcHdr := &greet.GreetServiceHandler{}
	greetProcedure, greetHandler := greetv1connect.NewGreetServiceHandler(greetSvcHdr)

	canvaSvcHdr := canva.NewCanvaServiceHandler()
	canvaProcedure, canvaHandler := canvav1connect.NewCanvaServiceHandler(canvaSvcHdr)

	// Register all handlers
	mux := http.NewServeMux()
	mux.Handle(fileProcedure, fileHandler)
	mux.Handle(greetProcedure, greetHandler)
	mux.Handle(canvaProcedure, canvaHandler)

	fawaSrv := &http.Server{
		Addr:    cfg.Addr,
		Handler: cors.NewCORS().Handler(mux),
	}

	// Setup graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		fwlog.Info("Shutting down server...")

		// Close canvas service
		canvaSvcHdr.Close()

		// Close file service
		if err := fileSvcHdr.Close(); err != nil {
			fwlog.Errorf("Error closing file service: %v", err)
		}

		// Set timeout for HTTP server shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := fawaSrv.Shutdown(ctx); err != nil {
			fwlog.Errorf("Server shutdown error: %v", err)
		}

		fwlog.Info("Server shutdown complete")
		os.Exit(0)
	}()

	fwlog.Infof("Server starting on %v", cfg.Addr)

	// Start the HTTPS server.
	if err := fawaSrv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fwlog.Fatalf("Failed to start server: %v", err)
	}
}
