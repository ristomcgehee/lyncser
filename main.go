package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alessio/shellescape"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	clientId     = "71296309757-nmhsm2ln7606lvgtoctqmo3ashvotfaa.apps.googleusercontent.com"
	clientSecret = "YmdfPYyJhlu08dHpVQQTnjbR"
)

type Config struct {
	Files    []string
	FilesAsk []string
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
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

func main() {

	data, err := ioutil.ReadFile("/home/chris/.config/go-syncer/config.json")
	checkError(err)
	var config Config
	err = json.Unmarshal(data, &config)
	checkError(err)
	fmt.Println(config)

	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	clientConfig, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(clientConfig)

	service, err := drive.New(client)
	checkError(err)
	listFilesCall := service.Files.List()
	listFilesCall.Fields("files(name, id, parents)")
	driveFileList, err := listFilesCall.Do()
	checkError(err)

	fmt.Println(driveFileList)

	goSyncerRoot := ""
	mapFiles := map[string]*drive.File{}
	for _, file := range driveFileList.Files {
		fmt.Println(file)
		fmt.Println(file.Name)
		if file.Name == "Go-Syncer-Root" {
			goSyncerRoot = file.Id
			continue
		}
		mapFiles[file.Id] = file
		if file.MimeType == "application/vnd.google-apps.folder" {

		}
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
	fmt.Println(mapPaths)

	for _, fileAsk := range config.FilesAsk {
		out, err := exec.Command("bash", "-c", "realpath "+shellescape.StripUnsafe(fileAsk)).Output()
		checkError(err)
		realPath := strings.TrimSpace(string(out[:]))
		fmt.Println(realPath)
		baseName := filepath.Base(fileAsk)
		dirId, ok := mapPaths[filepath.Dir(fileAsk)]
		fmt.Println(filepath.Dir(fileAsk))
		fmt.Println(dirId)
		fmt.Println(baseName)
		id, ok := mapPaths[fileAsk]
		if ok {
			fmt.Println(id)
		} else {
			f, err := os.Open(realPath)
			// f, err := os.Open("/home/chris/.bashrc")
			checkError(err)
			defer f.Close()

			// _, err = createDir(service, "config", "1KmrT8Yh_N8Ur0MXni__eGbC0_4-z_S5I")
			checkError(err)
			_, err = createFile(service, baseName, "text/plain", f, dirId)
			checkError(err)

			fmt.Printf("File '%s' successfully uploaded", fileAsk)
		}
	}

}

func createDir(service *drive.Service, name string, parentId string) (*drive.File, error) {
	d := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentId},
	}

	file, err := service.Files.Create(d).Do()

	if err != nil {
		log.Println("Could not create dir: " + err.Error())
		return nil, err
	}

	return file, nil
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
