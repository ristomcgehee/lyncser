package main

import (
	"fmt"

	"github.com/chrismcgehee/lyncser/sync"
	"github.com/chrismcgehee/lyncser/utils"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "lyncser",
}

func init() {
	rootCmd.AddCommand(
		&cobra.Command{
			Use: "sync",
			Short: "Syncs the files that are configured to be synced.",
			Run: syncCmd,
		},
	)
	deleteFilesCmd := &cobra.Command{
		Use: "deleteAllRemoteFiles",
		Short: "Deletes all files in the remote file store.",
		Run: deleteRemoteFiles,
	}
	deleteFilesCmd.Flags().BoolP("yes", "y", false, "Confirm deletion of all remote files")
	rootCmd.AddCommand(deleteFilesCmd)
}

func syncCmd(cmd *cobra.Command, args []string) {
	remoteFileStore, err := getRemoteFileStore()
	if err != nil {
		panic(err)
	}
	syncer := sync.Syncer{
		RemoteFileStore: remoteFileStore,
		LocalFileStore: &LocalFileStore{},
	}
	if err = syncer.PerformSync(); err != nil {
		panic(err)
	}
}

func deleteRemoteFiles(cmd *cobra.Command, args []string) {
	remoteFileStore, err := getRemoteFileStore()
	if err != nil {
		panic(err)
	}
	files, err := remoteFileStore.GetFiles()
	if err != nil {
		panic(err)
	}
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Printf("This will delete all %d files in the remote file store. Are you sure you want to continue? (y/n): ", len(files))
		var input string
		fmt.Scanln(&input)
		if input != "y" {
			return
		}
	}
	if err = remoteFileStore.DeleteAllFiles(); err != nil {
		panic(err)
	}
}

func getRemoteFileStore() (utils.FileStore, error) {
	encryptionKey, err := sync.GetEncryptionKey()
	if err != nil {
		return nil, err
	}
	encryptor := utils.AESGCMEncryptor{
		Key: encryptionKey,
	}
	return &DriveFileStore{
		Encryptor: encryptor,
	}, nil
}

func main() {
	rootCmd.Execute()
}
