package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

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
	clientConfig, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/drive.file")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(clientConfig)

	// conf := &oauth2.Config{
	// 	ClientID:     "",
	// 	ClientSecret: "",
	// 	Scopes:       []string{"https://www.googleapis.com/auth/drive.file"},
	// 	Endpoint:     google.Endpoint,
	// }
	// ctx := context.Background()
	// // Redirect user to consent page to ask for permission
	// // for the scopes specified above.
	// url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	// fmt.Printf("Visit the URL for the auth dialog: %v", url)

	// // Use the authorization code that is pushed to the redirect
	// // URL. Exchange will do the handshake to retrieve the
	// // initial access token. The HTTP Client returned by
	// // conf.Client will refresh the token as necessary.
	// var code string
	// _, err = fmt.Scan(&code)
	// checkError(err)
	// tok, err := conf.Exchange(ctx, code)
	// checkError(err)

	// client := conf.Client(ctx, tok)
	service, err := drive.New(client)
	checkError(err)
	driveFileList, err := service.Files.List().Do()

	fmt.Println(driveFileList)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
