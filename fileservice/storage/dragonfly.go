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
	"os"
	"time"

	"github.com/fawa-io/fawa/pkg/fwlog"
	"github.com/redis/go-redis/v9"
)

var dragon *DragonflyStorage

func init() {
	addr := os.Getenv("DRAGONFLY_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	dragon = &DragonflyStorage{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
			DB:   0,
		}),
	}
}

// DragonflyStorage implements the Storage interface using Dragonfly/Redis.
type DragonflyStorage struct {
	client redis.Cmdable
}

func (dragon *DragonflyStorage) saveFileMeta(key string, metadata *FileMetadata) error {
	if metadata == nil {
		return errors.New("metadata cannot be nil")
	}
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	ttl := 25 * time.Minute
	return dragon.client.Set(context.Background(), key, jsonMetadata, ttl).Err()
}

func (dragon *DragonflyStorage) getFileMeta(key string) (*FileMetadata, error) {
	val, err := dragon.client.Get(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}

	var metadata FileMetadata
	if err := json.Unmarshal([]byte(val), &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

func SaveFileMeta(key string, metadata *FileMetadata) error {
	return dragon.saveFileMeta(key, metadata)
}

func GetFileMeta(key string) (*FileMetadata, error) {
	return dragon.getFileMeta(key)
}

// Close closes storage connections
func Close() error {
	if dragon != nil {
		// Try to cast to redis.Client type
		if client, ok := dragon.client.(*redis.Client); ok {
			fwlog.Info("Closing Redis/Dragonfly connection...")
			return client.Close()
		}
		// Try to cast to redis.ClusterClient type
		if client, ok := dragon.client.(*redis.ClusterClient); ok {
			fwlog.Info("Closing Redis/Dragonfly cluster connection...")
			return client.Close()
		}
	}
	return nil
}
