package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/api/drive/v3"

	"github.com/chrismcgehee/lyncser/utils"
)

// File store that uses Google Drive.
type DriveFileStore struct {
	service *drive.Service
	// Key is the file's friendly name. Value is Google Drive file id. Contains an entry for each file/directory
	// in Google Drive that was created by lyncser.
	mapPathToFileId map[string]string
	// Key is Google Drive file id. Contains an entry for each file/directory in Google Drive that was created by
	// lyncser.
	mapIdToFile map[string]*drive.File
	// The Google Drive file id of the top-level folder where lyncser files are stored.
	lyncserRootId string
}

func (d *DriveFileStore) GetFiles() []utils.StoredFile {
	// This is the name of the top-level folder where all files created by lyncser will be stored.
	const lyncserRootName = "Lyncser-Root"
	d.service = getService(false)
	d.lyncserRootId = ""
	iface := makeApiCall(func() (interface{}, error) {
		fl, err := getFileList(d.service)
		return interface{}(fl), err
	}, d)
	fileList := iface.([]*drive.File)

	// Populate d.mapIdToFile and storedFiles with the files we got from the cloud.
	d.mapIdToFile = make(map[string]*drive.File)
	for _, file := range fileList {
		if file.Name == lyncserRootName {
			d.lyncserRootId = file.Id
			continue
		}
		d.mapIdToFile[file.Id] = file
	}

	if d.lyncserRootId == "" {
		iface = makeApiCall(func() (interface{}, error) {
			s, err := createDir(d.service, lyncserRootName, "")
			return interface{}(s), err
		}, d)
		d.lyncserRootId = iface.(string)
	}

	// Populate d.mapPathToFileId with the files that we can trace back to d.lyncserRootId
	d.mapPathToFileId = make(map[string]string)
	for id, file := range d.mapIdToFile {
		parentId := file.Parents[0]
		path := file.Name
		foundParent := false
		for true {
			if parentId == d.lyncserRootId {
				foundParent = true
				break
			}
			parentDir, ok := d.mapIdToFile[parentId]
			if !ok {
				// We can't find this file's parent. We'll act as if it doesn't exist in the cloud.
				break
			}
			foundParent = true
			path = parentDir.Name + "/" + path
			parentId = parentDir.Parents[0]
		}
		if foundParent {
			d.mapPathToFileId[path] = id
		}
	}


	storedFiles := make([]utils.StoredFile, 0, len(d.mapPathToFileId))
	for path, fileId := range d.mapPathToFileId {
		file := d.mapIdToFile[fileId]
		storedFiles = append(storedFiles, utils.StoredFile{
			Path: path,
			IsDir: file.MimeType == mimeTypeFolder,
		})
	}
	return storedFiles
}

// Creates this directory and any parent directories if they do not exist.
// Returns the Google Drive file id for the directory.
func (d *DriveFileStore) createDirIfNecessary(dirName string) string {
	if dirName == "" || dirName == "." || dirName == "/" {
		return d.lyncserRootId
	}
	dirId, ok := d.mapPathToFileId[dirName]
	if ok {
		return dirId // This directory already exists
	}
	parent := filepath.Dir(dirName)
	parentId, ok := d.mapPathToFileId[parent]
	if !ok {
		// The parent directory does not exist either. Recursively create it.
		parentId = d.createDirIfNecessary(parent)
	}
	iface := makeApiCall(func() (interface{}, error) {
		s, err := createDir(d.service, dirName, parentId)
		return interface{}(s), err
	}, d)
	dirId = iface.(string)
	d.mapPathToFileId[dirName] = dirId
	return dirId
}

func (d *DriveFileStore) CreateFile(file utils.SyncedFile) {
	baseName := filepath.Base(file.FriendlyPath)
	f, err := os.Open(file.RealPath)
	utils.PanicError(err)
	defer f.Close()

	dirId := d.createDirIfNecessary(filepath.Dir(file.FriendlyPath))
	makeApiCall(func() (interface{}, error) {
		f, err := createFile(d.service, baseName, "text/plain", f, dirId)
		return interface{}(f), err
	}, d)
}

func (d *DriveFileStore) GetModifiedTime(file utils.SyncedFile) time.Time {
	fileId := d.mapPathToFileId[file.FriendlyPath]
	driveFile := d.mapIdToFile[fileId]
	modTimeCloud, err := time.Parse(utils.TimeFormat, driveFile.ModifiedTime)
	utils.PanicError(err)
	return modTimeCloud
}

func (d *DriveFileStore) UpdateFile(file utils.SyncedFile) {
	fileId := d.mapPathToFileId[file.FriendlyPath]
	driveFile := d.mapIdToFile[fileId]
	f, err := os.Open(file.RealPath)
	utils.PanicError(err)
	makeApiCall(func() (interface{}, error) {
		f, err := updateFileContents(d.service, driveFile, fileId, f)
		return interface{}(f), err
	}, d)
}

func (d *DriveFileStore) DownloadFile(file utils.SyncedFile) {
	fileId := d.mapPathToFileId[file.FriendlyPath]
	iface := makeApiCall(func() (interface{}, error) {
		r, err := downloadFileContents(d.service, fileId)
		return interface{}(r), err
	}, d)
	contentsReader := iface.(io.ReadCloser)
	defer contentsReader.Close()
	dirName := filepath.Dir(file.RealPath)
	if !utils.PathExists(dirName) {
		os.MkdirAll(dirName, 0766)
	}
	out, err := os.OpenFile(file.RealPath, os.O_WRONLY|os.O_CREATE, 0644)
	utils.PanicError(err)
	defer out.Close()
	_, err = io.Copy(out, contentsReader)
	utils.PanicError(err)
}

func (d *DriveFileStore) FileExists(file utils.SyncedFile) bool {
	_, ok := d.mapPathToFileId[file.FriendlyPath]
	return ok
}

// Attempts an API call, and if it fails due to invalid token, will obtain a new one and try the API call again.
func makeApiCall(f func() (interface{}, error), d *DriveFileStore) interface{} {
	retval, err := f()
	if err != nil {
		if isTokenInvalid(err) {
			fmt.Println("Token is no longer valid. Requesting new one..")
			d.service = getService(true)
		}
		retval, err = f()
		utils.PanicError(err)
	}
	return retval
}

// To be re-introduced in Go 1.18.
// // Attempts an API call, and if it fails due to invalid token, will obtain a new one and try the API call again.
// func makeApiCall[T any](f func() (T, error), d *DriveFileStore) T {
// 	retval, err := f()
// 	if err != nil {
// 		if isTokenInvalid(err) {
// 			fmt.Println("Token is no longer valid. Requesting new one..")
// 			d.service = getService(true)
// 		}
// 		retval, err = f()
// 		utils.PanicError(err)
// 	}
// 	return retval
// }
