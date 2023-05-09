package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/ristomcgehee/lyncser/filestore"
	"github.com/ristomcgehee/lyncser/sync"
	"github.com/ristomcgehee/lyncser/utils"
)

const appVersion = "v0.1.20"

var rootCmd = &cobra.Command{
	Use: "lyncser",
}

//nolint:gochecknoinits
func init() {
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Syncs the files that are configured to be synced.",
		Run:   syncCmd,
	}
	addCommonFlags(syncCmd)
	syncCmd.Flags().BoolP("force-download", "f", false, "Forces download of all files")
	syncCmd.Flags().BoolP("dont-encrypt", "d", false, "Don't encrypt files. By default, files are encrypted.")
	rootCmd.AddCommand(syncCmd)
	deleteFilesCmd := &cobra.Command{
		Use:   "deleteAllRemoteFiles",
		Short: "Deletes all files in the remote file store.",
		Run:   deleteRemoteFiles,
	}
	addCommonFlags(deleteFilesCmd)
	deleteFilesCmd.Flags().BoolP("yes", "y", false, "Confirm deletion of all remote files")
	rootCmd.AddCommand(deleteFilesCmd)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of lyncser",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(appVersion)
		},
	}
	rootCmd.AddCommand(versionCmd)
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
	if err := cfg.Level.UnmarshalText([]byte(logLevel)); err != nil {
		return nil, err
	}
	cfg.Encoding = "console"
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()
	return sugar, nil
}

func syncCmd(cmd *cobra.Command, args []string) {
	logger, err := getLogger(cmd)
	if err != nil {
		panic(err)
	}
	forceDownload, err := cmd.Flags().GetBool("force-download")
	if err != nil {
		logger.Warn("error getting force-download flag", zap.Error(err))
	}
	remoteFileStore := getRemoteFileStore(logger)
	if err != nil {
		logger.Panic(err)
	}
	dontEncrypt, err := cmd.Flags().GetBool("dont-encrypt")
	if err != nil {
		logger.Warn("error getting dont-encrypt flag", zap.Error(err))
	}
	var encryptor utils.ReaderEncryptor
	if dontEncrypt {
		encryptor = &utils.NopEncryptor{}
	} else {
		encryptionKey, err := sync.GetEncryptionKey()
		if err != nil {
			logger.Panic(err)
		}
		encryptor = &utils.AESGCMEncryptor{
			Key: encryptionKey,
		}
	}
	syncer := sync.Syncer{
		RemoteFileStore: remoteFileStore,
		LocalFileStore:  &filestore.LocalFileStore{},
		Logger:          logger,
		Encryptor:       encryptor,
		ForceDownload:   forceDownload,
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
	remoteFileStore := getRemoteFileStore(logger)
	if err != nil {
		logger.Panic(err)
	}
	files, err := remoteFileStore.GetFiles()
	if err != nil {
		logger.Panic(err)
	}
	yes, err := cmd.Flags().GetBool("yes")
	if err != nil {
		logger.Warn("error getting yes flag", zap.Error(err))
	}
	if !yes {
		fmt.Printf("This will delete all %d files in the remote file store. Are you sure you want to continue? (y/n): ",
			len(files))
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

func getRemoteFileStore(logger utils.Logger) filestore.FileStore {
	return &filestore.DriveFileStore{
		Logger: logger,
	}
}

func main() {
	// Check for version flag before executing the root command
	if len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Println(appVersion)
		return
	}
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
