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
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/fawa-io/fawa/gen/fawa/canva/v1/canvav1connect"
	"github.com/fawa-io/fawa/gen/fawa/file/v1/filev1connect"
	"github.com/fawa-io/fawa/gen/fawa/greet/v1/greetv1connect"
	"github.com/fawa-io/fawa/pkg/cors"
	"github.com/fawa-io/fawa/pkg/fwlog"
	"github.com/fawa-io/fawa/pkg/util"
	"github.com/fawa-io/fawa/service/canva"
	"github.com/fawa-io/fawa/service/file"
	"github.com/fawa-io/fawa/service/greet"
)

var (
	addr      string
	uploadDir string
	certFile  string
	keyFile   string
)

func init() {
	flag.StringVar(&addr, "addr", "127.0.0.1:8080", "List of HTTP service address (e.g., '127.0.0.1:8080')")
	flag.StringVar(&uploadDir, "upload", "./upload", "Upload files dir")
	flag.StringVar(&certFile, "cert-file", "cert.pem", "Path to the TLS certificate file.")
	flag.StringVar(&keyFile, "key-file", "key.pem", "Path to the TLS private key file.")
}

func main() {
	// dev mode
	fwlog.SetLevel(fwlog.LevelDebug)

	flag.Parse()

	// Create upload dir for file service.
	if !util.Exist(uploadDir) {
		if err := util.CreateDir(uploadDir); err != nil {
			fwlog.Fatal(err)
		}
	}

	fileSvcHdr := &file.FileServiceHandler{
		UploadDir: uploadDir,
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
		Addr: addr,
		// Use h2c to handle gRPC requests over plain HTTP/2 (without TLS).
		Handler: h2c.NewHandler(cors.NewCORS().Handler(mux), &http2.Server{}),
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

	fwlog.Infof("Server starting on %v", addr)

	// Start the HTTPS server.
	if err := fawaSrv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
		fwlog.Fatalf("Failed to start server: %v", err)
	}
}
