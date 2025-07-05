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

package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"

	filev1 "github.com/fawa-io/fawa/gen/fawa/file/v1"
	"github.com/fawa-io/fawa/pkg/db"
	"github.com/fawa-io/fawa/pkg/fwlog"
	"github.com/fawa-io/fawa/pkg/util"
)

// FileServiceHandler implements the gRPC file service.
type FileServiceHandler struct {
	UploadDir string
}

// SendFile handles the client-streaming RPC to upload a file.
// The first message from the client must contain the file name,
// and subsequent messages contain the file's data chunks.
func (s *FileServiceHandler) SendFile(
	ctx context.Context,
	stream *connect.ClientStream[filev1.SendFileRequest],
) (*connect.Response[filev1.SendFileResponse], error) {
	fwlog.Info("SendFile request started")

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
	filePath := filepath.Join(s.UploadDir, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// processErr is a closure to handle file processing and ensure file.Close() is called.
	processErr := func() error {
		// Defer closing the file. If an error occurs during processing,
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

	strCtx := context.Background()
	downloadKey := util.Generaterandomstring(6)

	filesize, err := util.GetFileSize(filePath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	key := downloadKey
	metadata := db.FileMetadata{
		Filename:    fileName,
		Size:        filesize,
		StoragePath: filePath,
	}

	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		fmt.Println("JSON Marshal Failed:", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	expiration := 25 * time.Minute
	db.Dragonflydb.Set(strCtx, key, jsonMetadata, expiration)

	fwlog.Infof("File %s uploaded successfully.", fileName)
	res := connect.NewResponse(&filev1.SendFileResponse{
		Success: true,
		Message: "File " + fileName + " uploaded successfully.",
	})
	return res, nil
}

// ReceiveFile handles the server-streaming RPC to download a file.
// The client requests a file by name, and the server streams it back in chunks.
func (s *FileServiceHandler) ReceiveFile(
	ctx context.Context,
	req *connect.Request[filev1.ReceiveFileRequest],
	stream *connect.ServerStream[filev1.ReceiveFileResponse],
) (err error) {
	fileName := req.Msg.FileName

	fwlog.Debugf("Request to download file: %s", fileName)

	filePath := filepath.Join(s.UploadDir, fileName)
	file, err := os.Open(filePath)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, errors.New("file not found"))
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()

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
