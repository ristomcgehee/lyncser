package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const stateFilePath = "~/.config/go-syncer/state.json"
const configFilePath = "~/.config/go-syncer/config.json"
const credentialsFilePath = "~/.config/go-syncer/credentials.json"
const tokenFilePath = "~/.config/go-syncer/token.json"
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

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := realPath(tokenFilePath)
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func realPath(path string) string {
	out, err := exec.Command("bash", "-c", "realpath "+shellescape.StripUnsafe(path)).Output()
	checkError(err)
	return strings.TrimSpace(string(out[:]))
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

	b, err := ioutil.ReadFile(realPath(credentialsFilePath))
	checkError(err)

	// If modifying these scopes, delete your previously saved token.json.
	clientConfig, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	checkError(err)
	client := getClient(clientConfig)

	service, err := drive.New(client)
	checkError(err)
	listFilesCall := service.Files.List()
	listFilesCall.Fields("files(name, id, parents, modifiedTime)")
	driveFileList, err := listFilesCall.Do()
	checkError(err)

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
	// fmt.Println(stateData.FileStateData["~/.bashrc"])
}

func syncExistingFile(fileAsk string, fileId string, fileStats fs.FileInfo, mapFiles map[string]*drive.File, stateData *StateData, service *drive.Service) {
	realPath := realPath(fileAsk)
	driveFile := mapFiles[fileId]
	modTimeCloud, err := time.Parse(timeFormat, driveFile.ModifiedTime)
	checkError(err)
	var modTimeLocal time.Time
	modTimeLocal = fileStats.ModTime().UTC()
	lastCloudUpdate, err := time.Parse(timeFormat, stateData.FileStateData[fileAsk].LastCloudUpdate)
	checkError(err)

	if modTimeLocal.After(modTimeCloud) && modTimeLocal.After(lastCloudUpdate) {
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
	} else if modTimeCloud.After(lastCloudUpdate) {
		// Download from cloud
		fileGetCall := service.Files.Get(fileId)
		resp, err := fileGetCall.Download()
		checkError(err)
		defer resp.Body.Close()
		out, err := os.OpenFile(realPath, os.O_WRONLY|os.O_CREATE, 0644)
		checkError(err)
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		checkError(err)
		fmt.Printf("File '%s' successfully downloaded\n", fileAsk)
		stateData.FileStateData[fileAsk].LastCloudUpdate = time.Time.Format(time.Now().UTC(), timeFormat)
	}
}

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
		syncExistingFile(fileAsk, fileId, fileStats, mapFiles, stateData, service)
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

func createDir(service *drive.Service, name string, mapPaths map[string]string, goSyncerRoot string) string {
	if name == "" || name == "." || name == "/" {
		return goSyncerRoot
	}
	dirId, ok := mapPaths[name]
	if ok {
		return dirId // This directory already exists
	}
	parent := filepath.Dir(name)
	fmt.Println(parent)
	parentId, ok := mapPaths[parent]
	if !ok {
		// The parent directory does not exist either. Recursively create it.
		parentId = createDir(service, parent, mapPaths, goSyncerRoot)
	}
	d := &drive.File{
		Name:     filepath.Base(name),
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentId},
	}

	file, err := service.Files.Create(d).Do()
	checkError(err)
	fmt.Printf("Directory '%s' successfully uploaded\n", name)
	return file.Id
}

func createFile(service *drive.Service, name string, mimeType string, content io.Reader, parentId string) (*drive.File, error) {
	f := &drive.File{
		MimeType: mimeType,
		Name:     name,
		Parents:  []string{parentId},
	}
	file, err := service.Files.Create(f).Media(content).Do()

	if err != nil {
		log.Println("Could not create file: " + err.Error())
		return nil, err
	}

	return file, nil
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
