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
	"errors"
	"io"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	yaml "gopkg.in/yaml.v3"
)

// MinIOConfig holds the configuration for MinIO.
type MinIOConfig struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"accessKeyID"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	BucketName      string `yaml:"bucketName"`
	UseSSL          bool   `yaml:"useSSL"`
}

// minioFileStore holds the client and configuration for MinIO file operations.
type minioFileStore struct {
	client     *minio.Client
	bucketName string
}

var fileStore *minioFileStore

// init initializes the MinIO client and bucket from a YAML configuration file.
func init() {
	configPath := "pkg/storage/minio.yaml" // Assuming config.yaml is in the root directory
	configData, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Failed to read config file %s: %v, skipping MinIO client initialization.", configPath, err)
		return
	}

	var cfg MinIOConfig
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Failed to unmarshal config file %s: %v", configPath, err)
	}

	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.BucketName == "" {
		log.Println("MinIO configuration in config.yaml is incomplete, skipping client initialization.")
		return
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	fileStore = &minioFileStore{
		client:     client,
		bucketName: cfg.BucketName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		log.Fatalf("Failed to check if MinIO bucket '%s' exists: %v", cfg.BucketName, err)
	}
	if !exists {
		err = client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("Failed to create MinIO bucket '%s': %v", cfg.BucketName, err)
		}
		log.Printf("Successfully created MinIO bucket: %s", cfg.BucketName)
	}
}

// UploadFile uploads a file to MinIO.
// objectName is the full path/name of the object in the bucket.
// reader is the file content stream.
// size is the total size of the file.
func UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64) (minio.UploadInfo, error) {
	if fileStore == nil {
		return minio.UploadInfo{}, errors.New("MinIO client is not initialized")
	}

	return fileStore.client.PutObject(ctx, fileStore.bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream", // Generic content type
	})
}

// GetPresignedURL generates a temporary, presigned URL for downloading a file.
func GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	if fileStore == nil {
		return nil, errors.New("MinIO client is not initialized")
	}

	return fileStore.client.PresignedGetObject(ctx, fileStore.bucketName, objectName, expires, nil)
}
