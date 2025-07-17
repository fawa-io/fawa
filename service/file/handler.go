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
	"connectrpc.com/connect"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	filev1 "github.com/fawa-io/fawa/gen/fawa/file/v1"
	"github.com/fawa-io/fawa/pkg/fwlog"
	"github.com/fawa-io/fawa/pkg/storage"
	"github.com/fawa-io/fawa/pkg/util"
)

// FileServiceHandler implements the gRPC file service.
// It depends on a Storage interface for data persistence.
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
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing file info message"))
	}
	fileInfo := stream.Msg().GetInfo()
	fileName := stream.Msg().GetInfo().GetName()
	fileSize := stream.Msg().GetInfo().GetSize()
	if fileInfo == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("first message must be file info"))
	}

	// Security check to prevent path traversal attacks.
	if filepath.IsAbs(fileName) || strings.Contains(fileName, "..") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid file name"))
	}

	// Use a pipe to stream the file to MinIO without saving it to disk first.
	pr, pw := io.Pipe()

	var wg sync.WaitGroup
	wg.Add(1)
	errChan := make(chan error, 1)

	// Start a goroutine to upload the file to MinIO.
	go func() {
		defer wg.Done()
		defer func() {
			if err := pr.Close(); err != nil {
				fwlog.Errorf("Failed to close pipe reader: %v", err)
			}
		}()

		_, err := storage.UploadFile(ctx, fileName, pr, fileSize) // Use -1 for unknown size to stream

		if err != nil {
			errChan <- fmt.Errorf("minio upload failed: %w", err)
			fwlog.Errorf("Failed to upload file to MinIO: %v", err)
			return
		}
	}()

	// Stream data from the client to the pipe writer.
	processErr := func() (err error) {
		defer func() {
			if closeErr := pw.Close(); closeErr != nil {
				if err == nil {
					err = closeErr
				}
			}
		}()

		for stream.Receive() {
			chunk := stream.Msg().GetChunkData()
			if _, err := pw.Write(chunk); err != nil {
				return err
			}
		}
		return stream.Err()
	}()

	if processErr != nil {
		return nil, connect.NewError(connect.CodeInternal, processErr)
	}

	if err := stream.Err(); err != nil {
		if closeErr := pw.CloseWithError(fmt.Errorf("client stream error: %w", err)); closeErr != nil {
			log.Printf("Error closing pipe writer after client stream error: %v", closeErr)
		}
		wg.Wait()
		return nil, connect.NewError(connect.CodeAborted, err)
	}

	wg.Wait()
	close(errChan)
	if err := <-errChan; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	downloadKey := util.Generaterandomstring(6)

	// The fileSize is now determined by the upload info from MinIO.
	metadata := &storage.FileMetadata{
		Filename:    fileName,
		Size:        fileSize,
		StoragePath: fileName, // The storage path is now the object name in MinIO.
	}

	if err := storage.SaveFileMeta(downloadKey, metadata); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	fwlog.Infof("File %s uploaded successfully.", fileName)
	res := connect.NewResponse(&filev1.SendFileResponse{
		Success:   true,
		Message:   "File " + fileName + " uploaded successfully.",
		Randomkey: downloadKey,
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
	randomkey := req.Msg.Randomkey
	if randomkey == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("randomkey cannot be empty"))
	}

	metadata, err := storage.GetFileMeta(randomkey)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, errors.New("file not found"))
	}

	fileName := metadata.Filename
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
			Filename: fileName,
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

// GetDownloadURL generates a presigned URL for a file.
func (s *FileServiceHandler) GetDownloadURL(
	ctx context.Context,
	req *connect.Request[filev1.GetDownloadURLRequest],
) (*connect.Response[filev1.GetDownloadURLResponse], error) {
	randomkey := req.Msg.Randomkey
	if randomkey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("randomkey cannot be empty"))
	}

	metadata, err := storage.GetFileMeta(randomkey)
	if err != nil {
		// If the key is not found in your metadata store (e.g., Redis)
		fwlog.Error("Failed to get file metadata for key %s: %v", randomkey, err)
		return nil, connect.NewError(connect.CodeNotFound, errors.New("file not found or link expired"))
	}

	fwlog.Infof("Request to generate download URL for file: %s", metadata.StoragePath)

	expires := 5 * time.Minute
	presignedURL, err := storage.GetPresignedURL(ctx, metadata.StoragePath, expires)
	if err != nil {
		fwlog.Error("Failed to generate presigned URL for %s: %v", metadata.StoragePath, err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not generate download link"))
	}

	res := connect.NewResponse(&filev1.GetDownloadURLResponse{
		Url:      presignedURL.String(),
		Filename: metadata.Filename,
	})

	return res, nil
}
