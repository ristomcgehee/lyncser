package filestore

import (
	"io"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"

	"github.com/chrismcgehee/lyncser/utils"
)

// File store that uses Google Drive.
type DriveFileStore struct {
	Logger  utils.Logger
	service *drive.Service
	// Key is the file's friendly name. Value is Google Drive file id. Contains an entry for each file/directory
	// in Google Drive that was created by lyncser.
	mapPathToFileId map[string]string
	// Key is Google Drive file id. Contains an entry for each file/directory in Google Drive that was created by
	// lyncser.
	mapIdToFile map[string]*drive.File
	// The Google Drive file id of the top-level folder where lyncser files are stored.
	lyncserRootID string
}

func (d *DriveFileStore) GetFiles() ([]*StoredFile, error) {
	// This is the name of the top-level folder where all files created by lyncser will be stored.
	const lyncserRootName = "Lyncser-Root"
	var err error
	d.service, err = getService(false)
	if err != nil {
		return nil, err
	}
	d.lyncserRootID = ""
	fileList, err := getFileList(d.service)
	if err != nil {
		return nil, err
	}
	d.Logger.Debugf("Found %d files in Google Drive", len(fileList))

	// Populate d.mapIdToFile and storedFiles with the files we got from the cloud.
	d.mapIdToFile = make(map[string]*drive.File)
	for _, file := range fileList {
		if file.Name == lyncserRootName {
			d.lyncserRootID = file.Id
			continue
		}
		d.mapIdToFile[file.Id] = file
	}

	if d.lyncserRootID == "" {
		d.lyncserRootID, err = createDir(d.service, lyncserRootName, "")
		if err != nil {
			return nil, err
		}
		d.Logger.Debugf("New %s with id %s created", lyncserRootName, d.lyncserRootID)
	}

	// Populate d.mapPathToFileId with the files that we can trace back to d.lyncserRootID
	d.mapPathToFileId = make(map[string]string)
	for id, file := range d.mapIdToFile {
		parentId := file.Parents[0]
		path := file.Name
		foundParent := false
		for {
			if parentId == d.lyncserRootID {
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
			if !strings.HasPrefix(path, "~") {
				// When stored in Google Drive, file names do not start with '/'. We make up for that here.
				path = "/" + path
			}
			d.mapPathToFileId[path] = id
		}
	}

	storedFiles := make([]*StoredFile, 0, len(d.mapPathToFileId))
	for path, fileId := range d.mapPathToFileId {
		file := d.mapIdToFile[fileId]
		storedFiles = append(storedFiles, &StoredFile{
			Path:  path,
			IsDir: file.MimeType == mimeTypeFolder,
		})
	}
	return storedFiles, nil
}

func (d *DriveFileStore) GetFileContents(path string) (io.ReadCloser, error) {
	fileId, _ := d.getFiledID(path)
	return downloadFileContents(d.service, fileId)
}

func (d *DriveFileStore) GetModifiedTime(path string) (time.Time, error) {
	fileId, _ := d.getFiledID(path)
	driveFile := d.mapIdToFile[fileId]
	modTimeCloud, err := time.Parse(utils.TimeFormat, driveFile.ModifiedTime)
	if err != nil {
		return time.Now(), err
	}
	return modTimeCloud, nil
}

func (d *DriveFileStore) WriteFileContents(path string, reader io.Reader) error {
	fileId, exists := d.getFiledID(path)
	if !exists {
		d.createFile(path, reader)
		return nil
	}
	driveFile := d.mapIdToFile[fileId]
	_, err := updateFileContents(d.service, driveFile, fileId, reader)
	return err
}

func (d *DriveFileStore) DeleteFile(file string) error {
	fileId, exists := d.getFiledID(file)
	if !exists {
		return nil
	}
	return deleteFile(d.service, fileId)
}

func (d *DriveFileStore) DeleteAllFiles() error {
	return deleteFile(d.service, d.lyncserRootID)
}

func (d *DriveFileStore) FileExists(path string) (bool, error) {
	_, ok := d.getFiledID(path)
	return ok, nil
}

// getFiledID returns the Google Drive file id for the given path if it exists, otherwise it returns false for
// the second return value.
func (d *DriveFileStore) getFiledID(path string) (string, bool) {
	fileId, ok := d.mapPathToFileId[path]
	return fileId, ok
}

// Creates this directory and any parent directories if they do not exist.
// Returns the Google Drive file id for the directory.
func (d *DriveFileStore) createDirIfNecessary(dirName string) (string, error) {
	if dirName == "" || dirName == "." || dirName == "/" {
		return d.lyncserRootID, nil
	}
	dirId, ok := d.getFiledID(dirName)
	if ok {
		return dirId, nil // This directory already exists
	}
	var err error
	parent := filepath.Dir(dirName)
	parentID, ok := d.getFiledID(parent)
	if !ok {
		// The parent directory does not exist either. Recursively create it.
		parentID, err = d.createDirIfNecessary(parent)
		if err != nil {
			return "", err
		}
	}
	dirId, err = createDir(d.service, dirName, parentID)
	if err != nil {
		return "", err
	}
	d.Logger.Debugf("Directory '%s' successfully created", dirName)
	d.mapPathToFileId[dirName] = dirId
	return dirId, nil
}

func (d *DriveFileStore) createFile(path string, reader io.Reader) error {
	dirID, err := d.createDirIfNecessary(filepath.Dir(path))
	if err != nil {
		return err
	}
	baseName := filepath.Base(path)
	driveFile, err := createFile(d.service, baseName, "text/plain", reader, dirID)
	if err != nil {
		return err
	}
	d.mapPathToFileId[path] = driveFile.Id
	d.mapIdToFile[driveFile.Id] = driveFile
	return nil
}
