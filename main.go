package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"google.golang.org/api/drive/v3"
	yaml "gopkg.in/yaml.v3"
)

const (
	stateFilePath    = "~/.config/lyncser/state.json"
	globalConfigPath = "~/.config/lyncser/globalConfig.yaml"
	localConfigPath = "~/.config/lyncser/localConfig.yaml"
	timeFormat       = "2006-01-02T15:04:05.000Z"
	lyncserRootName  = "Lyncser-Root"
)

type GlobalConfig struct {
	TagFiles map[string][]string `yaml:"files"`
}

type LocalConfig struct {
	Tags []string `yaml:"tags"`
}

type StateData struct {
	FileStateData map[string]*FileStateData
}

type FileStateData struct {
	LastCloudUpdate string
}

func getGlobalConfig() GlobalConfig {
	fullConfigPath := realPath(globalConfigPath)
	data, err := ioutil.ReadFile(fullConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		configDir := path.Dir(fullConfigPath)
		os.MkdirAll(configDir, 0700)
		data = []byte("files:\n  all:\n    # - ~/.bashrc\n")
		err = os.WriteFile(fullConfigPath, data, 0644)
		checkError(err)
	} else {
		checkError(err)
	}
	var config GlobalConfig
	err = yaml.Unmarshal(data, &config)
	checkError(err)
	return config
}

func getLocalConfig() LocalConfig {
	fullConfigPath := realPath(localConfigPath)
	data, err := ioutil.ReadFile(fullConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		configDir := path.Dir(fullConfigPath)
		os.MkdirAll(configDir, 0700)
		data = []byte("tags:\n  - all\n")
		err = os.WriteFile(fullConfigPath, data, 0644)
		checkError(err)
	} else {
		checkError(err)
	}
	var config LocalConfig
	err = yaml.Unmarshal(data, &config)
	checkError(err)
	return config
}

func getStateData() StateData {
	var stateData StateData
	data, err := ioutil.ReadFile(realPath(stateFilePath))
	if errors.Is(err, os.ErrNotExist) {
		stateData = StateData{
			FileStateData: map[string]*FileStateData{},
		}
	} else {
		checkError(err)
		err = json.Unmarshal(data, &stateData)
		checkError(err)
	}
	return stateData
}

func saveStateData(stateData StateData) {
	data, err := json.MarshalIndent(stateData, "", " ")
	checkError(err)
	err = ioutil.WriteFile(realPath(stateFilePath), data, 0644)
	checkError(err)
}

func main() {
	globalConfig := getGlobalConfig()
	localConfig := getLocalConfig()
	stateData := getStateData()

	service := getService()
	driveFileList := getFileList(service)

	lyncserRoot := ""
	mapFiles := map[string]*drive.File{}
	for _, file := range driveFileList.Files {
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

	for tag, files := range globalConfig.TagFiles {
		if !inSlice(tag, localConfig.Tags) {
			continue
		}
		for _, fileName := range files {
			handleFile(fileName, mapPaths, mapFiles, &stateData, service, lyncserRoot)
		}
	}
	// globalConfigPath gets uploaded even if it's not explicitly listed
	handleFile(globalConfigPath, mapPaths, mapFiles, &stateData, service, lyncserRoot)

	saveStateData(stateData)
}

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
	fileExistsLocally := !errors.Is(err, os.ErrNotExist)
	if fileExistsLocally {
		checkError(err)
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
		checkError(err)
		defer f.Close()

		dirId = createDir(service, filepath.Dir(fileName), mapPaths, lyncserRoot)
		_, err = createFile(service, baseName, "text/plain", f, dirId)
		checkError(err)

		fmt.Printf("File '%s' successfully created\n", fileName)
		stateData.FileStateData[fileName].LastCloudUpdate = time.Time.Format(time.Now().UTC(), timeFormat)
	}
}

// Uploads/downloads the file as necessary
func syncExistingFile(fileName, fileId string, fileExistsLocally bool, fileStats fs.FileInfo,
	mapFiles map[string]*drive.File, stateData *StateData, service *drive.Service) {
	realPath := realPath(fileName)
	driveFile := mapFiles[fileId]
	modTimeCloud, err := time.Parse(timeFormat, driveFile.ModifiedTime)
	checkError(err)
	var modTimeLocal time.Time
	if fileExistsLocally {
		modTimeLocal = fileStats.ModTime().UTC()
	}
	lastCloudUpdate, err := time.Parse(timeFormat, stateData.FileStateData[fileName].LastCloudUpdate)
	checkError(err)

	if fileExistsLocally && modTimeLocal.After(modTimeCloud) && modTimeLocal.After(lastCloudUpdate) && lastCloudUpdate.Year() > 2001 {
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
		fmt.Printf("File '%s' successfully uploaded\n", fileName)
	} else if !fileExistsLocally || modTimeCloud.After(lastCloudUpdate) {
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
		fmt.Printf("File '%s' successfully downloaded\n", fileName)
	}
	stateData.FileStateData[fileName].LastCloudUpdate = time.Time.Format(time.Now().UTC(), timeFormat)
}
