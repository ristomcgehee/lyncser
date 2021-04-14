package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/api/drive/v3"
)

const stateFilePath = "~/.config/go-syncer/state.json"
const configFilePath = "~/.config/go-syncer/config.json"
const timeFormat = "2006-01-02T15:04:05.000Z"

type Config struct {
	Files    []string
	FilesAsk []string
}

type StateData struct {
	FileStateData map[string]*FileStateData
}

type FileStateData struct {
	LastCloudUpdate string
}

func main() {
	data, err := ioutil.ReadFile(realPath(configFilePath))
	checkError(err)
	var config Config
	err = json.Unmarshal(data, &config)
	checkError(err)

	data, err = ioutil.ReadFile(realPath(stateFilePath))
	checkError(err)
	var stateData StateData
	err = json.Unmarshal(data, &stateData)
	checkError(err)

	service := getService()
	driveFileList := getFileList(service)

	goSyncerRoot := ""
	mapFiles := map[string]*drive.File{}
	for _, file := range driveFileList.Files {
		if file.Name == "Go-Syncer-Root" {
			goSyncerRoot = file.Id
			continue
		}
		mapFiles[file.Id] = file
	}
	mapPaths := map[string]string{}
	for id, file := range mapFiles {
		parentId := file.Parents[0]
		path := file.Name
		for true {
			if parentId == goSyncerRoot {
				break
			}
			path = mapFiles[parentId].Name + "/" + path
			parentId = mapFiles[parentId].Parents[0]
		}
		mapPaths[path] = id
	}

	for _, fileAsk := range config.FilesAsk {
		handleFile(fileAsk, mapPaths, mapFiles, &stateData, service, goSyncerRoot)
	}
	data, err = json.MarshalIndent(stateData, "", " ")
	checkError(err)
	err = ioutil.WriteFile(realPath(stateFilePath), data, 0644)
	checkError(err)
}

// Creates the file if it does not exist in Google Drive, otherwise downloads or uploads the file to Google Drive
func handleFile(fileAsk string, mapPaths map[string]string, mapFiles map[string]*drive.File, stateData *StateData, service *drive.Service, goSyncerRoot string) {
	fmt.Println(fileAsk)
	realPath := realPath(fileAsk)
	fileStats, err := os.Stat(realPath)
	fileNotExists := errors.Is(err, os.ErrNotExist)
	if !fileNotExists {
		checkError(err)
	}
	if _, ok := stateData.FileStateData[fileAsk]; !ok {
		stateData.FileStateData[fileAsk] = &FileStateData{
			LastCloudUpdate: "2000-01-01T01:01:01.000Z",
		}
	}
	baseName := filepath.Base(fileAsk)
	dirId := mapPaths[filepath.Dir(fileAsk)]
	fileId, ok := mapPaths[fileAsk]
	if ok {
		syncExistingFile(fileAsk, fileId, !fileNotExists, fileStats, mapFiles, stateData, service)
	} else {
		if fileNotExists {
			return
		}
		f, err := os.Open(realPath)
		checkError(err)
		defer f.Close()

		dirId = createDir(service, filepath.Dir(fileAsk), mapPaths, goSyncerRoot)
		_, err = createFile(service, baseName, "text/plain", f, dirId)
		checkError(err)

		fmt.Printf("File '%s' successfully created\n", fileAsk)
		stateData.FileStateData[fileAsk].LastCloudUpdate = time.Time.Format(time.Now().UTC(), timeFormat)
	}
}

// Uploads/downloads the file as necessary
func syncExistingFile(fileAsk string, fileId string, fileExists bool, fileStats fs.FileInfo, mapFiles map[string]*drive.File, stateData *StateData, service *drive.Service) {
	realPath := realPath(fileAsk)
	driveFile := mapFiles[fileId]
	modTimeCloud, err := time.Parse(timeFormat, driveFile.ModifiedTime)
	checkError(err)
	var modTimeLocal time.Time
	if fileExists {
		modTimeLocal = fileStats.ModTime().UTC()
	}
	lastCloudUpdate, err := time.Parse(timeFormat, stateData.FileStateData[fileAsk].LastCloudUpdate)
	checkError(err)

	if fileExists && modTimeLocal.After(modTimeCloud) && modTimeLocal.After(lastCloudUpdate) {
		// Upload file to cloud
		f, err := os.Open(realPath)
		checkError(err)
		driveFile := &drive.File{
			MimeType: driveFile.MimeType,
			Name:     driveFile.Name,
		}
		fileUpdateCall := service.Files.Update(fileId, driveFile)
		fileUpdateCall.Media(f)
		_, err = fileUpdateCall.Do()
		checkError(err)
		fmt.Printf("File '%s' successfully uploaded\n", fileAsk)
		stateData.FileStateData[fileAsk].LastCloudUpdate = time.Time.Format(time.Now().UTC(), timeFormat)
	} else if !fileExists || modTimeCloud.After(lastCloudUpdate) {
		// Download from cloud
		fileGetCall := service.Files.Get(fileId)
		resp, err := fileGetCall.Download()
		checkError(err)
		defer resp.Body.Close()
		dirName := filepath.Dir(realPath)
		if !pathExists(dirName) {
			os.MkdirAll(dirName, 0766)
		}
		out, err := os.OpenFile(realPath, os.O_WRONLY|os.O_CREATE, 0644)
		checkError(err)
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		checkError(err)
		fmt.Printf("File '%s' successfully downloaded\n", fileAsk)
		stateData.FileStateData[fileAsk].LastCloudUpdate = time.Time.Format(time.Now().UTC(), timeFormat)
	}
}
