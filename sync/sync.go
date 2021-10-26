package sync

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chrismcgehee/lyncser/utils"
)

// PerformSync does the entire sync from end to end.
func PerformSync(fileStore utils.FileStore) {
	globalConfig := getGlobalConfig()
	localConfig := getLocalConfig()
	stateData := getStateData()

	fileStore.Initialize()

	for tag, paths := range globalConfig.TagPaths {
		if !inSlice(tag, localConfig.Tags) {
			continue
		}
		for _, pathToSync := range paths {
			realpath, err := filepath.EvalSymlinks(utils.RealPath(pathToSync))
			utils.PanicError(err)
			filepath.WalkDir(realpath, func(path string, d fs.DirEntry, err error) error {
				var pathError *fs.PathError
				if errors.As(err, &pathError) && pathError.Err.Error() != "no such file or directory" {
					utils.PanicError(err)
				}
				if d != nil && (d.IsDir() || !d.Type().IsRegular()) {
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
func handleFile(fileName string, stateData *StateData, fileStore utils.FileStore) {
	fmt.Println("Syncing", fileName)
	file := utils.SyncedFile{
		FriendlyPath: fileName,
		RealPath:     utils.RealPath(fileName),
	}
	fileStats, err := os.Stat(file.RealPath)
	fileExistsLocally := true
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fileExistsLocally = false
		} else {
			utils.PanicError(err)
		}
	}
	if _, ok := stateData.FileStateData[fileName]; !ok {
		stateData.FileStateData[fileName] = &FileStateData{
			LastCloudUpdate: "2000-01-01T01:01:01.000Z",
		}
	}
	if fileStore.FileExistsCloud(file) {
		syncExistingFile(file, fileExistsLocally, fileStats, stateData, fileStore)
		stateData.FileStateData[file.FriendlyPath].LastCloudUpdate = time.Time.Format(time.Now().UTC(), utils.TimeFormat)
	} else {
		if !fileExistsLocally {
			return
		}
		fileStore.CreateFile(file)
		fmt.Printf("File '%s' successfully created\n", file.FriendlyPath)
		stateData.FileStateData[file.FriendlyPath].LastCloudUpdate = time.Time.Format(time.Now().UTC(), utils.TimeFormat)
	}
}

// syncExistingFile uploads/downloads the file as necessary
func syncExistingFile(file utils.SyncedFile, fileExistsLocally bool, fileStats fs.FileInfo, stateData *StateData,
	fileStore utils.FileStore) {
	modTimeCloud := fileStore.GetCloudModifiedTime(file)
	var modTimeLocal time.Time
	if fileExistsLocally {
		modTimeLocal = fileStats.ModTime().UTC()
	}
	lastCloudUpdate, err := time.Parse(utils.TimeFormat, stateData.FileStateData[file.FriendlyPath].LastCloudUpdate)
	utils.PanicError(err)

	if fileExistsLocally && modTimeLocal.After(modTimeCloud) && modTimeLocal.After(lastCloudUpdate) && lastCloudUpdate.Year() > 2001 {
		fileStore.UpdateFile(file)
		fmt.Printf("File '%s' successfully uploaded\n", file.FriendlyPath)
	} else if !fileExistsLocally || modTimeCloud.After(lastCloudUpdate) {
		fileStore.DownloadFile(file)
		fmt.Printf("File '%s' successfully downloaded\n", file.FriendlyPath)
	}
}
