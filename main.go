package main

import (
	"github.com/chrismcgehee/lyncser/sync"
	"github.com/chrismcgehee/lyncser/utils"
)

func main() {
	encryptionKey, err := sync.GetEncryptionKey()
	if err != nil {
		panic(err)
	}
	encryptor := utils.AESGCMEncryptor{
		Key: encryptionKey,
	}
	syncer := sync.Syncer{
		RemoteFileStore: &DriveFileStore{
			Encryptor: encryptor,
		},
		LocalFileStore: &LocalFileStore{},
	}
	if err = syncer.PerformSync(); err != nil {
		panic(err)
	}
}
