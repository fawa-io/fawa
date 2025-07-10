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

package storage

// FileMetadata defines the structure for storing file information.
// This is the canonical definition used across the application.
type FileMetadata struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	StoragePath string `json:"storagePath"`
}

// Storage defines the interface for all data storage operations.
// This allows for decoupling the business logic from the concrete storage implementation.
type Storage interface {
	// SaveFileMeta saves the file metadata with a given key and TTL.
	SaveFileMeta(key string, metadata *FileMetadata) error

	// GetFileMeta retrieves file metadata by its key.
	GetFileMeta(key string) (*FileMetadata, error)
}
