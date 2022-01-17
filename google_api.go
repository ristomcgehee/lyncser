package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"

	"github.com/chrismcgehee/lyncser/utils"
)

const (
	// Path where OAuth client credentials are stored
	credentialsFilePath = "~/.config/lyncser/credentials.json"
	// Path where the OAuth token will be stored
	tokenFilePath = "~/.config/lyncser/token.json"
	// Mime type for files that are actually folders
	mimeTypeFolder = "application/vnd.google-apps.folder"
)

// getClient retrieves a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, forceNewToken bool) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile, err := utils.RealPath(tokenFilePath)
	if err != nil {
		return nil, err
	}
	var tok *oauth2.Token
	if !forceNewToken {
		tok, err = tokenFromFile(tokFile)
	}
	if err != nil || forceNewToken {
		tok, err = getTokenFromWeb(config)
		if err != nil {
			return nil, err
		}
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok), nil
}

// getTokenFromWeb requests a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, err
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, err
	}
	return tok, nil
}

// tokenFromFile retrieves a token from a local file.
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

// saveToken saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(token)
	return err
}

// getService returns a service that can be used to make API calls
func getService(forceNewToken bool) (*drive.Service, error) {
	realPath, err := utils.RealPath(credentialsFilePath)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadFile(realPath)
	if err != nil {
		return nil, err
	}

	// If modifying these scopes, delete the previously saved token.json.
	clientConfig, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return nil, err
	}
	client, err := getClient(clientConfig, forceNewToken)
	if err != nil {
		return nil, err
	}

	service, err := drive.New(client)
	if err != nil {
		return nil, err
	}
	return service, nil
}

// isTokenInvalid returns true if the error is for an invalid token.
func isTokenInvalid(err error) (bool, error) {
	var oauthError *oauth2.RetrieveError
	if errors.As(err, &oauthError) {
		r := struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}{}
		if err := json.Unmarshal(oauthError.Body, &r); err != io.EOF {
			return false, fmt.Errorf("error unmarshalling OAuth token: %w", err)
		}
		return r.Error == "invalid_grant", nil
	}
	return false, nil
}

// getFileList gets the list of file that this app has access to.
func getFileList(service *drive.Service) ([]*drive.File, error) {
	listFilesCall := service.Files.List()
	listFilesCall.Fields("files(name, id, parents, modifiedTime, mimeType), nextPageToken")
	listFilesCall.Q("trashed=false")
	var files []*drive.File
	for true {
		driveFileList, err := listFilesCall.Do()
		if err != nil {
			return nil, fmt.Errorf("error getting file list from Google Drive: %w", err)
		}
		files = append(files, driveFileList.Files...)
		if driveFileList.NextPageToken == "" {
			break
		}
		listFilesCall.PageToken(driveFileList.NextPageToken)
	}
	return files, nil
}

// createDir creates a directory in Google Drive. Returns the Id of the directory created.
func createDir(service *drive.Service, name, parentId string) (string, error) {
	d := &drive.File{
		Name:     filepath.Base(name),
		MimeType: mimeTypeFolder,
	}
	if parentId != "" {
		d.Parents = []string{parentId}
	}

	file, err := service.Files.Create(d).Do()
	if err != nil {
		return "", fmt.Errorf("error creating directory in Google Drive: %w", err)
	}
	return file.Id, nil
}

// createFile creates the file in Google Drive.
func createFile(service *drive.Service, name string, mimeType string, content io.Reader, parentId string) (*drive.File, error) {
	f := &drive.File{
		MimeType: mimeType,
		Name:     name,
		Parents:  []string{parentId},
	}
	file, err := service.Files.Create(f).Media(content).Do()
	if err != nil {
		return nil, fmt.Errorf("error creating file in Google Drive: %w", err)
	}
	return file, nil
}

// downloadFileContents returns the contents of the file as an io.ReadCloser.
func downloadFileContents(service *drive.Service, fileId string) (io.ReadCloser, error) {
	fileGetCall := service.Files.Get(fileId)
	resp, err := fileGetCall.Download()
	if err != nil {
		return nil, fmt.Errorf("error downloading file contents from Google Drive: %w", err)
	}
	return resp.Body, nil
}

// updateFileContents uploads the contents from the io.Reader.
func updateFileContents(service *drive.Service, driveFile *drive.File, fileId string, r io.Reader) (*drive.File, error) {
	driveFile = &drive.File{
		MimeType: driveFile.MimeType,
		Name:     driveFile.Name,
	}
	fileUpdateCall := service.Files.Update(fileId, driveFile)
	fileUpdateCall.Media(r)
	file, err := fileUpdateCall.Do()
	if err != nil {
		return nil, fmt.Errorf("error updating file contents from Google Drive: %w", err)
	}
	return file, nil
}

// deleteFile deletes the file in Google Drive.
func deleteFile(service *drive.Service, fileId string) error {
	fileDeleteCall := service.Files.Delete(fileId)
	if err := fileDeleteCall.Do(); err != nil {
		return fmt.Errorf("error deleting file from Google Drive: %w", err)
	}
	return nil
}
