package main

import (
	"fmt"

	"github.com/chrismcgehee/lyncser/sync"
	"github.com/chrismcgehee/lyncser/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use: "lyncser",
}

func init() {
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Syncs the files that are configured to be synced.",
		Run:   syncCmd,
	}
	addCommonFlags(syncCmd)
	rootCmd.AddCommand(syncCmd)
	deleteFilesCmd := &cobra.Command{
		Use:   "deleteAllRemoteFiles",
		Short: "Deletes all files in the remote file store.",
		Run:   deleteRemoteFiles,
	}
	addCommonFlags(deleteFilesCmd)
	deleteFilesCmd.Flags().BoolP("yes", "y", false, "Confirm deletion of all remote files")
	rootCmd.AddCommand(deleteFilesCmd)
}

func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("log-level", "l", "info", "The log level to use. One of: debug, info, warn, error, fatal")
}

func getLogger(cmd *cobra.Command) (*zap.SugaredLogger, error) {
	cfg := zap.NewProductionConfig()
	logLevel, err := cmd.Flags().GetString("log-level")
	if err != nil {
		return nil, err
	}
	cfg.Level.UnmarshalText([]byte(logLevel))
	cfg.Encoding = "console"
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()
	return sugar, nil
}

func syncCmd(cmd *cobra.Command, args []string) {
	logger, err := getLogger(cmd)
	if err != nil {
		panic(err)
	}
	remoteFileStore, err := getRemoteFileStore(logger)
	if err != nil {
		logger.Panic(err)
	}
	encryptionKey, err := sync.GetEncryptionKey()
	if err != nil {
		logger.Panic(err)
	}
	encryptor := &utils.AESGCMEncryptor{
		Key: encryptionKey,
	}
	syncer := sync.Syncer{
		RemoteFileStore: remoteFileStore,
		LocalFileStore:  &LocalFileStore{},
		Logger:          logger,
		Encryptor:       encryptor,
	}
	if err = syncer.PerformSync(); err != nil {
		logger.Panic(err)
	}
}

func deleteRemoteFiles(cmd *cobra.Command, args []string) {
	logger, err := getLogger(cmd)
	if err != nil {
		panic(err)
	}
	remoteFileStore, err := getRemoteFileStore(logger)
	if err != nil {
		logger.Panic(err)
	}
	files, err := remoteFileStore.GetFiles()
	if err != nil {
		logger.Panic(err)
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
		logger.Panic(err)
	}
	logger.Infof("Deleted %d files", len(files))
}

func getRemoteFileStore(logger utils.Logger) (utils.FileStore, error) {
	return &DriveFileStore{
		Logger:    logger,
	}, nil
}

func main() {
	rootCmd.Execute()
}
