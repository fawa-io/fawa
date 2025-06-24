package util

import (
	"github.com/redis/go-redis/v9"
)

var Redis_dragonfly = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "",
	DB:       0,
})

type FileMetadata struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	StoragePath string `json:"storagePath"`
}
