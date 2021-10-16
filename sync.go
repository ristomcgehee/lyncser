package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
)

const (
	timeFormat      = "2006-01-02T15:04:05.000Z"
	lyncserRootName = "Lyncser-Root"
)

// performSync does the entire sync from end to end.
func performSync() {
	globalConfig := getGlobalConfig()
	localConfig := getLocalConfig()
	stateData := getStateData()

	service := getService()
	driveFiles := getFileList(service)

	lyncserRoot := ""
	mapFiles := map[string]*drive.File{}
	for _, file := range driveFiles {
		if file.Name == lyncserRootName {
			lyncserRoot = file.Id
			continue
		}
		mapFiles[file.Id] = file
	}

	mapPaths := map[string]string{}
	if lyncserRoot == "" {
		lyncserRoot = createDir(service, lyncserRootName, mapPaths, lyncserRoot)
	}

	for id, file := range mapFiles {
		parentId := file.Parents[0]
		path := file.Name
		for true {
			if parentId == lyncserRoot {
				break
			}
			path = mapFiles[parentId].Name + "/" + path
			parentId = mapFiles[parentId].Parents[0]
		}
		mapPaths[path] = id
	}

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
				handleFile(path, mapPaths, mapFiles, &stateData, service, lyncserRoot)
				return nil
			})
		}
	}
	// globalConfigPath gets uploaded even if it's not explicitly listed
	handleFile(globalConfigPath, mapPaths, mapFiles, &stateData, service, lyncserRoot)

	saveStateData(stateData)
}

// inSlice returns true if item is presint in slice.
func inSlice(item string, slice []string) bool {
	for _, sliceItem := range slice {
		if item == sliceItem {
			return true
		}
	}
	return false
}

// Creates the file if it does not exist in Google Drive, otherwise downloads or uploads the file to Google Drive
func handleFile(fileName string, mapPaths map[string]string, mapFiles map[string]*drive.File, stateData *StateData,
	service *drive.Service, lyncserRoot string) {
	fmt.Println(fileName)
	realPath := realPath(fileName)
	fileStats, err := os.Stat(realPath)
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
	baseName := filepath.Base(fileName)
	dirId := mapPaths[filepath.Dir(fileName)]
	fileId, fileExistsCloud := mapPaths[fileName]
	if fileExistsCloud {
		syncExistingFile(fileName, fileId, fileExistsLocally, fileStats, mapFiles, stateData, service)
	} else {
		if !fileExistsLocally {
			return
		}
		f, err := os.Open(realPath)
		panicError(err)
		defer f.Close()

		dirId = createDir(service, filepath.Dir(fileName), mapPaths, lyncserRoot)
		_, err = createFile(service, baseName, "text/plain", f, dirId)
		panicError(err)

		fmt.Printf("File '%s' successfully created\n", fileName)
		stateData.FileStateData[fileName].LastCloudUpdate = time.Time.Format(time.Now().UTC(), timeFormat)
	}
}

// syncExistingFile uploads/downloads the file as necessary
func syncExistingFile(fileName, fileId string, fileExistsLocally bool, fileStats fs.FileInfo,
	mapFiles map[string]*drive.File, stateData *StateData, service *drive.Service) {
	realPath := realPath(fileName)
	driveFile := mapFiles[fileId]
	modTimeCloud, err := time.Parse(timeFormat, driveFile.ModifiedTime)
	panicError(err)
	var modTimeLocal time.Time
	if fileExistsLocally {
		modTimeLocal = fileStats.ModTime().UTC()
	}
	lastCloudUpdate, err := time.Parse(timeFormat, stateData.FileStateData[fileName].LastCloudUpdate)
	panicError(err)

	if fileExistsLocally && modTimeLocal.After(modTimeCloud) && modTimeLocal.After(lastCloudUpdate) && lastCloudUpdate.Year() > 2001 {
		// Upload file to cloud
		f, err := os.Open(realPath)
		panicError(err)
		driveFile := &drive.File{
			MimeType: driveFile.MimeType,
			Name:     driveFile.Name,
		}
		fileUpdateCall := service.Files.Update(fileId, driveFile)
		fileUpdateCall.Media(f)
		_, err = fileUpdateCall.Do()
		panicError(err)
		fmt.Printf("File '%s' successfully uploaded\n", fileName)
	} else if !fileExistsLocally || modTimeCloud.After(lastCloudUpdate) {
		// Download from cloud
		fileGetCall := service.Files.Get(fileId)
		resp, err := fileGetCall.Download()
		panicError(err)
		defer resp.Body.Close()
		dirName := filepath.Dir(realPath)
		if !pathExists(dirName) {
			os.MkdirAll(dirName, 0766)
		}
		out, err := os.OpenFile(realPath, os.O_WRONLY|os.O_CREATE, 0644)
		panicError(err)
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		panicError(err)
		fmt.Printf("File '%s' successfully downloaded\n", fileName)
	}
	stateData.FileStateData[fileName].LastCloudUpdate = time.Time.Format(time.Now().UTC(), timeFormat)
}
