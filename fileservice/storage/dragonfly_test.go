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
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

func TestDragonflyStorage_SaveFileMeta(t *testing.T) {
	client, mock := redismock.NewClientMock()

	storage := &DragonflyStorage{client: client}

	testCases := []struct {
		name     string
		key      string
		metadata *FileMetadata
		mocker   func()
		wantErr  bool
	}{
		{
			name: "success",
			key:  "test-key",
			metadata: &FileMetadata{
				Filename:    "test.txt",
				Size:        123,
				StoragePath: "/path/to/file",
			},
			mocker: func() {
				metadataJSON, _ := json.Marshal(&FileMetadata{
					Filename:    "test.txt",
					Size:        123,
					StoragePath: "/path/to/file",
				})
				mock.ExpectSet("test-key", metadataJSON, 25*time.Minute).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name:     "nil metadata",
			key:      "nil-key",
			metadata: nil,
			mocker:   func() {},
			wantErr:  true,
		},
		{
			name: "redis error",
			key:  "error-key",
			metadata: &FileMetadata{
				Filename: "error.txt",
			},
			mocker: func() {
				metadataJSON, _ := json.Marshal(&FileMetadata{Filename: "error.txt"})
				mock.ExpectSet("error-key", metadataJSON, 25*time.Minute).SetErr(errors.New("redis error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mocker()
			err := storage.saveFileMeta(tc.key, tc.metadata)
			if (err != nil) != tc.wantErr {
				t.Errorf("SaveFileMeta() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestDragonflyStorage_GetFileMeta(t *testing.T) {
	client, mock := redismock.NewClientMock()

	storage := &DragonflyStorage{client: client}

	testCases := []struct {
		name       string
		key        string
		mocker     func()
		wantResult *FileMetadata
		wantErr    bool
	}{
		{
			name: "success",
			key:  "test-key",
			mocker: func() {
				metadata := &FileMetadata{
					Filename:    "test.txt",
					Size:        123,
					StoragePath: "/path/to/file",
				}
				metadataJSON, _ := json.Marshal(metadata)
				mock.ExpectGet("test-key").SetVal(string(metadataJSON))
			},
			wantResult: &FileMetadata{
				Filename:    "test.txt",
				Size:        123,
				StoragePath: "/path/to/file",
			},
			wantErr: false,
		},
		{
			name: "key not found",
			key:  "not-found-key",
			mocker: func() {
				mock.ExpectGet("not-found-key").SetErr(redis.Nil)
			},
			wantResult: nil,
			wantErr:    true,
		},
		{
			name: "json unmarshal error",
			key:  "invalid-json-key",
			mocker: func() {
				mock.ExpectGet("invalid-json-key").SetVal("invalid json")
			},
			wantResult: nil,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mocker()
			got, err := storage.getFileMeta(tc.key)
			if (err != nil) != tc.wantErr {
				t.Errorf("GetFileMeta() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tc.wantResult) {
				t.Errorf("GetFileMeta() got = %v, want %v", got, tc.wantResult)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

// setupRealDragonfly creates a real client and skips tests if the service is unavailable.
func setupRealDragonfly(b *testing.B) *DragonflyStorage {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Default address for Dragonfly/Redis
		DB:   0,
	})

	// Check if the connection is alive
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("skipping benchmark: cannot connect to Dragonfly/Redis on localhost:6379. Error: %v", err)
	}

	return &DragonflyStorage{client: client}
}

func BenchmarkGetFileMeta(b *testing.B) {
	storage := setupRealDragonfly(b)
	if b.Skipped() {
		return
	}

	metadata := &FileMetadata{
		Filename:    "benchmark.txt",
		Size:        1024,
		StoragePath: "/benchmark/path",
	}
	// Pre-populate data for the benchmark to fetch.
	key := "benchmark-get-key"
	err := storage.saveFileMeta(key, metadata)
	if err != nil {
		b.Fatalf("failed to set up benchmark data: %v", err)
	}

	// Low Concurrency (Sequential)
	b.Run("Low-Concurrency-1", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := storage.getFileMeta(key)
			if err != nil {
				b.Error(err)
			}
		}
	})

	// Medium Concurrency (CPUs / 2)
	if medProcs := runtime.NumCPU() / 2; medProcs > 1 {
		b.Run(fmt.Sprintf("Medium-Concurrency-%d", medProcs), func(b *testing.B) {
			prevProcs := runtime.GOMAXPROCS(medProcs)
			defer runtime.GOMAXPROCS(prevProcs)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, err := storage.getFileMeta(key)
					if err != nil {
						b.Error(err)
					}
				}
			})
		})
	}

	// High Concurrency (Default GOMAXPROCS)
	b.Run(fmt.Sprintf("High-Concurrency-%d", runtime.NumCPU()), func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := storage.getFileMeta(key)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})

	// Very High Concurrency (CPUs * 2)
	veryHighProcs := runtime.NumCPU() * 2
	b.Run(fmt.Sprintf("VeryHigh-Concurrency-%d", veryHighProcs), func(b *testing.B) {
		prevProcs := runtime.GOMAXPROCS(veryHighProcs)
		defer runtime.GOMAXPROCS(prevProcs)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := storage.getFileMeta(key)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})
}

func BenchmarkSaveFileMeta(b *testing.B) {
	storage := setupRealDragonfly(b)
	if b.Skipped() {
		return
	}
	metadata := &FileMetadata{
		Filename:    "benchmark.txt",
		Size:        1024,
		StoragePath: "/benchmark/path",
	}
	key := "benchmark-save-key"

	// Low Concurrency (Sequential)
	b.Run("Low-Concurrency-1", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := storage.saveFileMeta(key, metadata)
			if err != nil {
				b.Error(err)
			}
		}
	})

	// Medium Concurrency (CPUs / 2)
	if medProcs := runtime.NumCPU() / 2; medProcs > 1 {
		b.Run(fmt.Sprintf("Medium-Concurrency-%d", medProcs), func(b *testing.B) {
			prevProcs := runtime.GOMAXPROCS(medProcs)
			defer runtime.GOMAXPROCS(prevProcs)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := storage.saveFileMeta(key, metadata)
					if err != nil {
						b.Error(err)
					}
				}
			})
		})
	}

	// High Concurrency (Default GOMAXPROCS)
	b.Run(fmt.Sprintf("High-Concurrency-%d", runtime.NumCPU()), func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := storage.saveFileMeta(key, metadata)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})

	// Very High Concurrency (CPUs * 2)
	veryHighProcs := runtime.NumCPU() * 2
	b.Run(fmt.Sprintf("VeryHigh-Concurrency-%d", veryHighProcs), func(b *testing.B) {
		prevProcs := runtime.GOMAXPROCS(veryHighProcs)
		defer runtime.GOMAXPROCS(prevProcs)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := storage.saveFileMeta(key, metadata)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})
}
