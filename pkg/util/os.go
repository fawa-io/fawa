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

package util

import (
	"os"
)

const (
	// the owner can make/remove files inside the directory
	privateDirMode = 0700
)

func Exist(dirpath string) bool {
	names, err := readDir(dirpath)
	if err != nil {
		return false
	}
	return len(names) != 0
}

// readDir returns the filenames in a directory.
func readDir(dirpath string) ([]string, error) {
	dir, err := os.Open(dirpath)
	if err != nil {
		return nil, err
	}

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	err = dir.Close()
	if err != nil {
		return nil, err
	}

	return names, nil
}

func CreateDir(dirpath string) error {
	if Exist(dirpath) {
		return os.ErrExist
	}

	if err := os.MkdirAll(dirpath, privateDirMode); err != nil {
		return err
	}

	return nil
}

func GetFileSize(dirpath string) (int64, error) {
	fileInfo, err := os.Stat(dirpath)
	if err != nil {
		return 0, err
	}

	fileSize := fileInfo.Size()
	return fileSize, nil
}
