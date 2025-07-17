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
)

// minioFileStore holds the client and configuration for MinIO file operations.
type minioFileStore struct {
	client     *minio.Client
	bucketName string
}

var fileStore *minioFileStore

// init initializes the MinIO client and bucket from environment variables.
func init() {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("MINIO_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("MINIO_BUCKET_NAME")
	useSSL := os.Getenv("MINIO_USE_SSL") == "true"

	log.Printf("Initializing MinIO with the following configuration:")
	log.Printf("  MINIO_ENDPOINT: %s", endpoint)
	log.Printf("  MINIO_ACCESS_KEY_ID: %s", accessKeyID)
	log.Printf("  MINIO_BUCKET_NAME: %s", bucketName)
	log.Printf("  MINIO_USE_SSL: %v", useSSL)

	if endpoint == "" || accessKeyID == "" || secretAccessKey == "" || bucketName == "" {
		log.Println("MinIO environment variables for file storage not set, skipping client initialization.")
		return
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	fileStore = &minioFileStore{
		client:     client,
		bucketName: bucketName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		log.Fatalf("Failed to check if MinIO bucket '%s' exists: %v", bucketName, err)
	}
	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("Failed to create MinIO bucket '%s': %v", bucketName, err)
		}
		log.Printf("Successfully created MinIO bucket: %s", bucketName)
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

// ListObjects lists all objects in the bucket for debugging purposes.
func ListObjects(ctx context.Context) ([]string, error) {
	if fileStore == nil {
		return nil, errors.New("MinIO client is not initialized")
	}

	var objectNames []string
	objectCh := fileStore.client.ListObjects(ctx, fileStore.bucketName, minio.ListObjectsOptions{})
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		objectNames = append(objectNames, object.Key)
	}
	return objectNames, nil
}
