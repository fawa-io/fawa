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
	"log"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"connectrpc.com/connect"

	filev1 "github.com/fawa-io/fawa/gen/file/v1"
	"github.com/fawa-io/fawa/gen/file/v1/filev1connect"
)

const (
	uploadDir = "./uploads"
)

type fileServiceHandler struct{}

// SendFile
func (s *fileServiceHandler) SendFile(
	ctx context.Context,
	stream *connect.ClientStream[filev1.SendFileRequest],
) (*connect.Response[filev1.SendFileResponse], error) {
	log.Println("SendFile request started")

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !stream.Receive() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing file name"))
	}
	fileName := stream.Msg().GetFileName()
	if fileName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("file name cannot be empty"))
	}
	log.Printf("Receiving file: %s", fileName)

	//create
	filePath := filepath.Join(uploadDir, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer file.Close()

	// get file chunk
	for stream.Receive() {
		chunk := stream.Msg().GetChunkData()
		if _, err := file.Write(chunk); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := stream.Err(); err != nil {
		return nil, connect.NewError(connect.CodeUnknown, err)
	}

	log.Printf("File %s uploaded successfully.", fileName)
	res := connect.NewResponse(&filev1.SendFileResponse{
		Success: true,
		Message: "File " + fileName + " uploaded successfully.",
	})
	return res, nil
}

// ReceiveFile
func (s *fileServiceHandler) ReceiveFile(
	ctx context.Context,
	req *connect.Request[filev1.ReceiveFileRequest],
	stream *connect.ServerStream[filev1.ReceiveFileResponse],
) error {
	fileName := req.Msg.FileName
	log.Printf("Request to download file: %s", fileName)
	filePath := filepath.Join(uploadDir, fileName)

	file, err := os.Open(filePath)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, errors.New("file not found"))
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if err := stream.Send(&filev1.ReceiveFileResponse{
		Payload: &filev1.ReceiveFileResponse_FileSize{
			FileSize: fileInfo.Size(),
		},
	}); err != nil {
		return err
	}

	buffer := make([]byte, 1024*64) // 64KB buffer
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}

		if err := stream.Send(&filev1.ReceiveFileResponse{
			Payload: &filev1.ReceiveFileResponse_ChunkData{
				ChunkData: buffer[:n],
			},
		}); err != nil {
			return err
		}
	}

	log.Printf("File %s sent successfully.", fileName)
	return nil
}

func main() {
	// make sure dir exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("failed to create upload directory: %v", err)
	}

	mux := http.NewServeMux()
	path, handler := filev1connect.NewFileServiceHandler(&fileServiceHandler{})
	mux.Handle(path, handler)

	log.Println("Server starting on :8080...")
	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: h2c.NewHandler(mux, &http2.Server{}), // no tls now
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
