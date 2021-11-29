package utils

import "time"

type FileStore interface {
	// GetFiles returns the list of file that are stored in this file store.
	GetFiles() ([]StoredFile, error)
	CreateFile(path SyncedFile) error
	UpdateFile(path SyncedFile) error
	DownloadFile(path SyncedFile) error
	GetModifiedTime(path SyncedFile) (time.Time, error)
	FileExists(path SyncedFile) (bool, error)
}

type StoredFile struct {
	Path  string
	IsDir bool
}

type SyncedFile struct {
	FriendlyPath string
	RealPath     string
}
