package main

import (
	// "github.com/chrismcgehee/lyncser/file_store"
	"github.com/chrismcgehee/lyncser/sync"
)

func main() {
	fileStore := &DriveFileStore{}
	sync.PerformSync(fileStore)
}
