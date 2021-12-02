package main

import (
	"github.com/chrismcgehee/lyncser/sync"
)

func main() {
	syncer := sync.Syncer{
		RemoteFileStore: &DriveFileStore{},
		LocalFileStore: &LocalFileStore{},
	}
	if err := syncer.PerformSync(); err != nil {
		panic(err)
	}
}
