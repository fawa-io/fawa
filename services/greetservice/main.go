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

	"github.com/fawa-io/fawa/services/greetservice/config"
	"github.com/fawa-io/fawa/services/greetservice/gen/greet/v1/greetv1connect"
	greet "github.com/fawa-io/fawa/services/greetservice/handler"
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

	greetSvcHdr := &greet.GreetServiceHandler{}
	greetProcedure, greetHandler := greetv1connect.NewGreetServiceHandler(greetSvcHdr)

	mux := http.NewServeMux()
	mux.Handle(greetProcedure, greetHandler)

	greetSrv := &http.Server{
		Addr:    cfg.Addr,
		Handler: cors.NewCORS().Handler(mux),
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		fwlog.Info("Shutting down server...")

		// Set timeout for HTTP server shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := greetSrv.Shutdown(ctx); err != nil {
			fwlog.Errorf("Server shutdown error: %v", err)
		}

		fwlog.Info("Server shutdown complete")
		os.Exit(0)
	}()

	fwlog.Infof("Server starting on %v", cfg.Addr)

	// Start the HTTPS server.
	if err := greetSrv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fwlog.Fatalf("Failed to start server: %v", err)
	}
}
