package utils

import "time"

type FileStore interface {
	Initialize()
	CreateFile(path SyncedFile)
	UpdateFile(path SyncedFile)
	DownloadFile(path SyncedFile)
	GetModifiedTime(path SyncedFile) time.Time
	FileExists(path SyncedFile) bool
}

type SyncedFile struct {
	FriendlyPath string
	RealPath     string
}
