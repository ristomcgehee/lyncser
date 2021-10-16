package main

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/api/drive/v3"
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

func (d *DriveFileStore) initialize() {
	// This is the name of the top-level folder where all files created by lyncser will be stored.
	const lyncserRootName = "Lyncser-Root"
	d.service = getService()
	d.lyncserRootId = ""
	fileList := getFileList(d.service)
	d.mapIdToFile = make(map[string]*drive.File)
	for _, file := range fileList {
		if file.Name == lyncserRootName {
			d.lyncserRootId = file.Id
			continue
		}
		d.mapIdToFile[file.Id] = file
	}

	d.mapPathToFileId = make(map[string]string)
	if d.lyncserRootId == "" {
		d.lyncserRootId = createDir(d.service, lyncserRootName, "")
	}

	for id, file := range d.mapIdToFile {
		parentId := file.Parents[0]
		path := file.Name
		for true {
			if parentId == d.lyncserRootId {
				break
			}
			path = d.mapIdToFile[parentId].Name + "/" + path
			parentId = d.mapIdToFile[parentId].Parents[0]
		}
		d.mapPathToFileId[path] = id
	}
}

// creates this directory and any parent directories if they do not exist.
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
	dirId = createDir(d.service, dirName, parentId)
	d.mapPathToFileId[dirName] = dirId
	return dirId
}

func (d *DriveFileStore) createFile(file SyncedFile) {
	baseName := filepath.Base(file.friendlyPath)
	f, err := os.Open(file.realPath)
	panicError(err)
	defer f.Close()

	dirId := d.createDirIfNecessary(filepath.Dir(file.friendlyPath))
	_, err = createFile(d.service, baseName, "text/plain", f, dirId)
	panicError(err)
}

func (d *DriveFileStore) getCloudModifiedTime(file SyncedFile) time.Time {
	fileId := d.mapPathToFileId[file.friendlyPath]
	driveFile := d.mapIdToFile[fileId]
	modTimeCloud, err := time.Parse(timeFormat, driveFile.ModifiedTime)
	panicError(err)
	return modTimeCloud
}

func (d *DriveFileStore) updateFile(file SyncedFile) {
	fileId := d.mapPathToFileId[file.friendlyPath]
	driveFile := d.mapIdToFile[fileId]
	f, err := os.Open(file.realPath)
	panicError(err)
	driveFile = &drive.File{
		MimeType: driveFile.MimeType,
		Name:     driveFile.Name,
	}
	fileUpdateCall := d.service.Files.Update(fileId, driveFile)
	fileUpdateCall.Media(f)
	_, err = fileUpdateCall.Do()
	panicError(err)
}

func (d *DriveFileStore) downloadFile(file SyncedFile) {
	fileId := d.mapPathToFileId[file.friendlyPath]
	fileGetCall := d.service.Files.Get(fileId)
	resp, err := fileGetCall.Download()
	panicError(err)
	defer resp.Body.Close()
	dirName := filepath.Dir(file.realPath)
	if !pathExists(dirName) {
		os.MkdirAll(dirName, 0766)
	}
	out, err := os.OpenFile(file.realPath, os.O_WRONLY|os.O_CREATE, 0644)
	panicError(err)
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	panicError(err)
}

func (d *DriveFileStore) fileExistsCloud(path SyncedFile) bool {
	_, ok := d.mapPathToFileId[path.friendlyPath]
	return ok
}
