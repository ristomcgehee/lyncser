package utils

import "time"

type FileStore interface {
	Initialize()
	CreateFile(path SyncedFile)
	UpdateFile(path SyncedFile)
	DownloadFile(path SyncedFile)
	GetCloudModifiedTime(path SyncedFile) time.Time
	FileExistsCloud(path SyncedFile) bool
}

type SyncedFile struct {
	FriendlyPath string
	RealPath     string
}
