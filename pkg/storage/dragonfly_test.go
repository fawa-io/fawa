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
	"encoding/json"
	"errors"
	"reflect"
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
			err := storage.SaveFileMeta(tc.key, tc.metadata)
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
			got, err := storage.GetFileMeta(tc.key)
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