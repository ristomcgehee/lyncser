package main

import (
	"github.com/chrismcgehee/lyncser/sync"
)

func main() {
	syncer := sync.Syncer{
		RemoteFileStore: &DriveFileStore{},
		LocalFileStore: &LocalFileStore{},
	}
	err := syncer.PerformSync()
	if err != nil {
		panic(err)
	}
}
