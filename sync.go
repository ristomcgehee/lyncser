package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	timeFormat = "2006-01-02T15:04:05.000Z"
)

type FileStore interface {
	initialize()
	createFile(path SyncedFile)
	updateFile(path SyncedFile)
	downloadFile(path SyncedFile)
	getCloudModifiedTime(path SyncedFile) time.Time
	fileExistsCloud(path SyncedFile) bool
}

type SyncedFile struct {
	friendlyPath string
	realPath     string
}

// performSync does the entire sync from end to end.
func performSync() {
	globalConfig := getGlobalConfig()
	localConfig := getLocalConfig()
	stateData := getStateData()

	fileStore := &DriveFileStore{}
	fileStore.initialize()

	for tag, paths := range globalConfig.TagPaths {
		if !inSlice(tag, localConfig.Tags) {
			continue
		}
		for _, pathToSync := range paths {
			realpath := realPath(pathToSync)
			filepath.WalkDir(realpath, func(path string, d fs.DirEntry, err error) error {
				panicError(err)
				if d.IsDir() {
					return nil
				}
				path = strings.Replace(path, realpath, pathToSync, 1)
				handleFile(path, &stateData, fileStore)
				return nil
			})
		}
	}
	// globalConfigPath gets uploaded even if it's not explicitly listed
	handleFile(globalConfigPath, &stateData, fileStore)

	saveStateData(stateData)
}

// inSlice returns true if item is present in slice.
func inSlice(item string, slice []string) bool {
	for _, sliceItem := range slice {
		if item == sliceItem {
			return true
		}
	}
	return false
}

// Creates the file if it does not exist in the cloud, otherwise downloads or uploads the file to the cloud
func handleFile(fileName string, stateData *StateData, fileStore FileStore) {
	file := SyncedFile{
		friendlyPath: fileName,
		realPath:     realPath(fileName),
	}
	fmt.Println("Syncing", fileName)
	fileStats, err := os.Stat(file.realPath)
	fileExistsLocally := true
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fileExistsLocally = false
		} else {
			panicError(err)
		}
	}
	if _, ok := stateData.FileStateData[fileName]; !ok {
		stateData.FileStateData[fileName] = &FileStateData{
			LastCloudUpdate: "2000-01-01T01:01:01.000Z",
		}
	}
	if fileStore.fileExistsCloud(file) {
		syncExistingFile(file, fileExistsLocally, fileStats, stateData, fileStore)
		stateData.FileStateData[file.friendlyPath].LastCloudUpdate = time.Time.Format(time.Now().UTC(), timeFormat)
	} else {
		if !fileExistsLocally {
			return
		}
		fileStore.createFile(file)
		fmt.Printf("File '%s' successfully created\n", file.friendlyPath)
		stateData.FileStateData[file.friendlyPath].LastCloudUpdate = time.Time.Format(time.Now().UTC(), timeFormat)
	}
}

// syncExistingFile uploads/downloads the file as necessary
func syncExistingFile(file SyncedFile, fileExistsLocally bool, fileStats fs.FileInfo, stateData *StateData,
	fileStore FileStore) {
	modTimeCloud := fileStore.getCloudModifiedTime(file)
	var modTimeLocal time.Time
	if fileExistsLocally {
		modTimeLocal = fileStats.ModTime().UTC()
	}
	lastCloudUpdate, err := time.Parse(timeFormat, stateData.FileStateData[file.friendlyPath].LastCloudUpdate)
	panicError(err)

	if fileExistsLocally && modTimeLocal.After(modTimeCloud) && modTimeLocal.After(lastCloudUpdate) && lastCloudUpdate.Year() > 2001 {
		fileStore.updateFile(file)
		fmt.Printf("File '%s' successfully uploaded\n", file.friendlyPath)
	} else if !fileExistsLocally || modTimeCloud.After(lastCloudUpdate) {
		fileStore.downloadFile(file)
		fmt.Printf("File '%s' successfully downloaded\n", file.friendlyPath)
	}
}
