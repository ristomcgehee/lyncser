package main

import (
	"errors"
	"os"
	"time"

	"github.com/chrismcgehee/lyncser/utils"
)

// For accessing local files.
type LocalFileStore struct{}

func (l *LocalFileStore) Initialize() {
}

// Creates this directory and any parent directories if they do not exist.
// Returns the Google Drive file id for the directory.
func (l *LocalFileStore) createDirIfNecessary(dirName string) string {
	panic("not implemented")
}

func (l *LocalFileStore) CreateFile(file utils.SyncedFile) {
	panic("not implemented")
}

func (l *LocalFileStore) GetModifiedTime(file utils.SyncedFile) time.Time {
	fileStats, err := os.Stat(file.RealPath)
	utils.PanicError(err)
	return fileStats.ModTime()
}

func (l *LocalFileStore) UpdateFile(file utils.SyncedFile) {
	panic("not implemented")
}

func (l *LocalFileStore) DownloadFile(file utils.SyncedFile) {
	panic("not implemented")
}

func (l *LocalFileStore) FileExists(file utils.SyncedFile) bool {
	fileExistsLocally := true
	_, err := os.Stat(file.RealPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fileExistsLocally = false
		} else {
			utils.PanicError(err)
		}
	}
	return fileExistsLocally
}
