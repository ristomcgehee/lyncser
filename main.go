package main

import (
	"github.com/chrismcgehee/lyncser/sync"
)

func main() {
	syncer := sync.Syncer{
		RemoteFileStore: &DriveFileStore{},
		LocalFileStore: &LocalFileStore{},
	}
	syncer.PerformSync()
}
