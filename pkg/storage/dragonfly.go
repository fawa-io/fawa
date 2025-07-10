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

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// DragonflyStorage implements the Storage interface using Dragonfly/Redis.
type DragonflyStorage struct {
	client redis.Cmdable
}

// NewDragonflyStorage creates a new instance of DragonflyStorage.
// It returns a Storage interface, hiding the implementation details.
func NewDragonflyStorage(addr string) (Storage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	// Check the connection.
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &DragonflyStorage{client: client}, nil
}

// SaveFileMetadata implements the Storage interface.
func (d *DragonflyStorage) SaveFileMeta(key string, metadata *FileMetadata) error {
	if metadata == nil {
		return errors.New("metadata cannot be nil")
	}
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	ttl := 25 * time.Minute
	return d.client.Set(context.Background(), key, jsonMetadata, ttl).Err()
}

// GetFileMetadata implements the Storage interface.
func (d *DragonflyStorage) GetFileMeta(key string) (*FileMetadata, error) {
	val, err := d.client.Get(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}

	var metadata FileMetadata
	if err := json.Unmarshal([]byte(val), &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}
