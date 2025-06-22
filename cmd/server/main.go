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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"connectrpc.com/connect"

	filev1 "github.com/fawa-io/fawa/gen/fawa/file/v1"
	"github.com/fawa-io/fawa/gen/fawa/file/v1/filev1connect"
	"github.com/fawa-io/fawa/pkg/cors"
	"github.com/fawa-io/fawa/pkg/fwlog"
)

const (
	// uploadDir is the directory where uploaded files are stored.
	uploadDir = "./uploads"
)

// fileServiceHandler implements the gRPC file service.
type fileServiceHandler struct{}

// SendFile handles the client-streaming RPC to upload a file.
// The first message from the client must contain the file name,
// and subsequent messages contain the file's data chunks.
func (s *fileServiceHandler) SendFile(
	ctx context.Context,
	stream *connect.ClientStream[filev1.SendFileRequest],
) (*connect.Response[filev1.SendFileResponse], error) {
	fwlog.Info("SendFile request started")
	// Ensure the upload directory exists.
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// The first message is expected to contain metadata (file name).
	if !stream.Receive() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing file name"))
	}
	fileName := stream.Msg().GetFileName()
	if fileName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("file name cannot be empty"))
	}
	fwlog.Infof("Receiving file: %s", fileName)

	// Security check to prevent path traversal attacks.
	if filepath.IsAbs(fileName) || strings.Contains(fileName, "..") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid file name"))
	}

	// Create the file on the server.
	filePath := filepath.Join(uploadDir, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// processErr is a closure to handle file processing and ensure file.Close() is called.
	processErr := func() error {
		// Defer closing the file. If an error occurs during processing,
		// this will capture the close error if one occurs.
		defer func() {
			if closeErr := file.Close(); err == nil {
				err = closeErr
			}
		}()

		// Receive file data chunks in a loop.
		for stream.Receive() {
			chunk := stream.Msg().GetChunkData()
			if _, err := file.Write(chunk); err != nil {
				return err // Return write error.
			}
		}
		return stream.Err() // Return any error from the stream itself.
	}()

	if processErr != nil {
		return nil, connect.NewError(connect.CodeInternal, processErr)
	}

	fwlog.Infof("File %s uploaded successfully.", fileName)
	res := connect.NewResponse(&filev1.SendFileResponse{
		Success: true,
		Message: "File " + fileName + " uploaded successfully.",
	})
	return res, nil
}

// ReceiveFile handles the server-streaming RPC to download a file.
// The client requests a file by name, and the server streams it back in chunks.
func (s *fileServiceHandler) ReceiveFile(
	ctx context.Context,
	req *connect.Request[filev1.ReceiveFileRequest],
	stream *connect.ServerStream[filev1.ReceiveFileResponse],
) (err error) {
	fileName := req.Msg.FileName
	fwlog.Info("Request to download file: %s", fileName)
	filePath := filepath.Join(uploadDir, fileName)

	// Open the requested file.
	file, err := os.Open(filePath)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, errors.New("file not found"))
	}
	// Ensure the file is closed upon function exit.
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()
	fwlog.Info("Request to download file: %s", fileName)
	// Get file info to send the size first.
	fileInfo, err := file.Stat()
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	// Send file size as the first message in the stream.
	if err := stream.Send(&filev1.ReceiveFileResponse{
		Payload: &filev1.ReceiveFileResponse_FileSize{
			FileSize: fileInfo.Size(),
		},
	}); err != nil {
		return err
	}

	// Stream the file content in chunks.
	buffer := make([]byte, 1024*64) // 64KB buffer
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break // End of file reached.
		}
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}

		// Send a data chunk.
		if err := stream.Send(&filev1.ReceiveFileResponse{
			Payload: &filev1.ReceiveFileResponse_ChunkData{
				ChunkData: buffer[:n],
			},
		}); err != nil {
			return err
		}
	}

	fwlog.Infof("File %s sent successfully.", fileName)
	return nil
}

func main() {
	// Ensure upload directory exists on startup.
	// TODO: move to pkg
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fwlog.Fatalf("failed to create upload directory: %v", err)
	}

	mux := http.NewServeMux()
	// Create a new handler for the FileService.
	procedure, handler := filev1connect.NewFileServiceHandler(&fileServiceHandler{})
	// Register the handler with the mux.
	mux.Handle(procedure, handler)

	fwlog.Infof("Server starting on :8080...")
	fawaSrv := &http.Server{
		Addr: "localhost:8080",
		// Use h2c to handle gRPC requests over plain HTTP/2 (without TLS).
		Handler: h2c.NewHandler(cors.NewCORS().Handler(mux), &http2.Server{}),
	}
	// Start the HTTP server.
	err := fawaSrv.ListenAndServe()
	if err != nil {
		fwlog.Fatalf("failed to serve: %v", err)
	}
}
