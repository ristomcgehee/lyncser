package main

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/chrismcgehee/lyncser/utils"
)

// For accessing local files.
type LocalFileStore struct{}

func (l *LocalFileStore) GetFiles() ([]utils.StoredFile, error) {
	panic("not implemented")
}

func (l *LocalFileStore) GetFileContents(file utils.SyncedFile) (io.ReadCloser, error) {
	return os.Open(file.RealPath)
}

func (l *LocalFileStore) GetModifiedTime(file utils.SyncedFile) (time.Time, error) {
	fileStats, err := os.Stat(file.RealPath)
	if err != nil {
		return time.Now(), err
	}
	return fileStats.ModTime(), nil
}

func (l *LocalFileStore) WriteFileContents(file utils.SyncedFile, contentReader io.Reader) error {
	dirName := filepath.Dir(file.RealPath)
	pathExists, err := utils.PathExists(dirName)
	if err != nil {
		return err
	}
	if !pathExists {
		err = os.MkdirAll(dirName, 0766)
		if err != nil {
			return err
		}
	}
	out, err := os.OpenFile(file.RealPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, contentReader); err != nil {
		return err
	}
	return nil
}

func (l *LocalFileStore) DeleteFile(file string) error {
	panic("not implemented")
}

func (l *LocalFileStore) FileExists(file utils.SyncedFile) (bool, error) {
	return utils.PathExists(file.RealPath)
}
