package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"

	"github.com/chrismcgehee/lyncser/utils"
)

// File store that uses Google Drive.
type DriveFileStore struct {
	// Used to encrypt files stored in Google Drive.
	Encryptor utils.AESGCMEncryptor
	service   *drive.Service
	// Key is the file's friendly name. Value is Google Drive file id. Contains an entry for each file/directory
	// in Google Drive that was created by lyncser.
	mapPathToFileId map[string]string
	// Key is Google Drive file id. Contains an entry for each file/directory in Google Drive that was created by
	// lyncser.
	mapIdToFile map[string]*drive.File
	// The Google Drive file id of the top-level folder where lyncser files are stored.
	lyncserRootId string
}

func (d *DriveFileStore) GetFiles() ([]utils.StoredFile, error) {
	// This is the name of the top-level folder where all files created by lyncser will be stored.
	const lyncserRootName = "Lyncser-Root"
	var err error
	d.service, err = getService(false)
	if err != nil {
		return nil, err
	}
	d.lyncserRootId = ""
	iface, err := makeApiCall(func() (interface{}, error) {
		fl, err := getFileList(d.service)
		return interface{}(fl), err
	}, d)
	if err != nil {
		return nil, err
	}
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
		iface, err = makeApiCall(func() (interface{}, error) {
			s, err := createDir(d.service, lyncserRootName, "")
			return interface{}(s), err
		}, d)
		if err != nil {
			return nil, err
		}
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
			Path:  path,
			IsDir: file.MimeType == mimeTypeFolder,
		})
	}
	return storedFiles, nil
}

func (d *DriveFileStore) GetFileContents(path string) (io.ReadCloser, error) {
	fileId, _ := d.getFiledId(path)
	iface, err := makeApiCall(func() (interface{}, error) {
		r, err := downloadFileContents(d.service, fileId)
		return interface{}(r), err
	}, d)
	if err != nil {
		return nil, err
	}
	contentsReader := iface.(io.ReadCloser)
	defer contentsReader.Close()

	decryptedReader, err := d.Encryptor.DecryptReader(contentsReader)
	if err != nil {
		return nil, err
	}

	return decryptedReader, nil
}

func (d *DriveFileStore) GetModifiedTime(path string) (time.Time, error) {
	fileId, _ := d.getFiledId(path)
	driveFile := d.mapIdToFile[fileId]
	modTimeCloud, err := time.Parse(utils.TimeFormat, driveFile.ModifiedTime)
	if err != nil {
		return time.Now(), err
	}
	return modTimeCloud, nil
}

func (d *DriveFileStore) WriteFileContents(path string, reader io.Reader) error {
	readerEncrypted, err := d.Encryptor.EncryptReader(reader)
	if err != nil {
		return err
	}

	fileId, exists := d.getFiledId(path)
	if !exists {
		d.createFile(path, readerEncrypted)
		return nil
	}
	driveFile := d.mapIdToFile[fileId]
	_, err = makeApiCall(func() (interface{}, error) {
		f, err := updateFileContents(d.service, driveFile, fileId, readerEncrypted)
		return interface{}(f), err
	}, d)
	return err
}

func (d *DriveFileStore) DeleteFile(file string) error {
	fileId, exists := d.getFiledId(file)
	if !exists {
		return nil
	}
	_, err := makeApiCall(func() (interface{}, error) {
		err := deleteFile(d.service, fileId)
		return nil, err
	}, d)
	return err
}

func (d *DriveFileStore) FileExists(path string) (bool, error) {
	_, ok := d.getFiledId(path)
	return ok, nil
}

// getFiledId returns the Google Drive file id for the given path if it exists, otherwise it returns false for
// the second return value.
func (d *DriveFileStore) getFiledId(path string) (string, bool) {
	// When stored in Google Drive, file names do not start with '/'.
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	fileId, ok := d.mapPathToFileId[path]
	return fileId, ok
}

// Creates this directory and any parent directories if they do not exist.
// Returns the Google Drive file id for the directory.
func (d *DriveFileStore) createDirIfNecessary(dirName string) (string, error) {
	if dirName == "" || dirName == "." || dirName == "/" {
		return d.lyncserRootId, nil
	}
	dirId, ok := d.getFiledId(dirName)
	if ok {
		return dirId, nil // This directory already exists
	}
	var err error
	parent := filepath.Dir(dirName)
	parentId, ok := d.getFiledId(parent)
	if !ok {
		// The parent directory does not exist either. Recursively create it.
		parentId, err = d.createDirIfNecessary(parent)
		if err != nil {
			return "", err
		}
	}
	iface, err := makeApiCall(func() (interface{}, error) {
		s, err := createDir(d.service, dirName, parentId)
		return interface{}(s), err
	}, d)
	if err != nil {
		return "", err
	}
	dirId = iface.(string)
	d.mapPathToFileId[dirName] = dirId
	return dirId, nil
}

func (d *DriveFileStore) createFile(path string, reader io.Reader) error {
	dirId, err := d.createDirIfNecessary(filepath.Dir(path))
	if err != nil {
		return err
	}
	baseName := filepath.Base(path)
	iface, err := makeApiCall(func() (interface{}, error) {
		f, err := createFile(d.service, baseName, "text/plain", reader, dirId)
		return interface{}(f), err
	}, d)
	if err != nil {
		return err
	}
	driveFile := iface.(*drive.File)
	d.mapPathToFileId[path] = driveFile.Id
	d.mapIdToFile[driveFile.Id] = driveFile
	return nil
}

// Attempts an API call, and if it fails due to invalid token, will obtain a new one and try the API call again.
func makeApiCall(f func() (interface{}, error), d *DriveFileStore) (interface{}, error) {
	retval, err := f()
	if err != nil {
		isTokenInvalid, err := isTokenInvalid(err)
		if err != nil {
			return nil, err
		}
		if isTokenInvalid {
			fmt.Println("Token is no longer valid. Requesting new one..")
			d.service, err = getService(true)
			if err != nil {
				return nil, err
			}
		}
		retval, err = f()
		if err != nil {
			return nil, err
		}
	}
	return retval, err
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
