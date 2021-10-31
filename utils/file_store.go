package utils

import "time"

type FileStore interface {
	// GetFiles returns the list of file that are stored in this file store.
	GetFiles() []StoredFile
	CreateFile(path SyncedFile)
	UpdateFile(path SyncedFile)
	DownloadFile(path SyncedFile)
	GetModifiedTime(path SyncedFile) time.Time
	FileExists(path SyncedFile) bool
}

type StoredFile struct {
	Path  string
	IsDir bool
}

type SyncedFile struct {
	FriendlyPath string
	RealPath     string
}
