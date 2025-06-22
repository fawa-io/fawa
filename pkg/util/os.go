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
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
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
