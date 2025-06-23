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
	"flag"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/fawa-io/fawa/gen/fawa/file/v1/filev1connect"
	"github.com/fawa-io/fawa/pkg/cors"
	"github.com/fawa-io/fawa/pkg/fwlog"
	"github.com/fawa-io/fawa/pkg/util"
	"github.com/fawa-io/fawa/service/file"
)

var (
	addr      string
	uploadDir string
)

func init() {
	flag.StringVar(&addr, "addr", "127.0.0.1:8080", "List of HTTP service address (e.g., '127.0.0.1:8080')")
	flag.StringVar(&uploadDir, "upload", "./uploads", "Uploads files dir")
}

func main() {
	flag.Parse()

	if !util.Exist(uploadDir) {
		if err := util.CreateDir(uploadDir); err != nil {
			fwlog.Fatal(err)
		}
	}

	// Create a new handler for the FileService.
	fileSvcHdr := &file.FileServiceHandler{
		UploadDir: uploadDir,
	}
	procedure, handler := filev1connect.NewFileServiceHandler(fileSvcHdr)

	mux := http.NewServeMux()
	mux.Handle(procedure, handler)

	fawaSrv := &http.Server{
		Addr: addr,
		// Use h2c to handle gRPC requests over plain HTTP/2 (without TLS).
		Handler: h2c.NewHandler(cors.NewCORS().Handler(mux), &http2.Server{}),
	}

	fwlog.Infof("Server starting on %v", addr)

	// Start the HTTP server.
	err := fawaSrv.ListenAndServe()
	if err != nil {
		fwlog.Fatalf("failed to serve: %v", err)
	}
}
