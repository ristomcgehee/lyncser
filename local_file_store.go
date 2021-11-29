package main

import (
	"errors"
	"os"
	"time"

	"github.com/chrismcgehee/lyncser/utils"
)

// For accessing local files.
type LocalFileStore struct{}

func (l *LocalFileStore) GetFiles() ([]utils.StoredFile, error) {
	panic("not implemented")
}

func (l *LocalFileStore) CreateFile(file utils.SyncedFile) error {
	panic("not implemented")
}

func (l *LocalFileStore) GetModifiedTime(file utils.SyncedFile) (time.Time, error) {
	fileStats, err := os.Stat(file.RealPath)
	if err != nil {
		return time.Now(), err
	}
	return fileStats.ModTime(), nil
}

func (l *LocalFileStore) UpdateFile(file utils.SyncedFile) error {
	panic("not implemented")
}

func (l *LocalFileStore) DownloadFile(file utils.SyncedFile) error {
	panic("not implemented")
}

func (l *LocalFileStore) FileExists(file utils.SyncedFile) (bool, error) {
	fileExistsLocally := true
	_, err := os.Stat(file.RealPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fileExistsLocally = false
		} else {
			return false, err
		}
	}
	return fileExistsLocally, nil
}
