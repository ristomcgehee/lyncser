package file_store

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
	lyncserRootId string
}

func (d *DriveFileStore) GetFiles() ([]*StoredFile, error) {
	// This is the name of the top-level folder where all files created by lyncser will be stored.
	const lyncserRootName = "Lyncser-Root"
	var err error
	d.service, err = getService(false)
	if err != nil {
		return nil, err
	}
	d.lyncserRootId = ""
	fileList, err := getFileList(d.service)
	if err != nil {
		return nil, err
	}
	d.Logger.Debugf("Found %d files in Google Drive", len(fileList))

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
		d.lyncserRootId, err = createDir(d.service, lyncserRootName, "")
		if err != nil {
			return nil, err
		}
		d.Logger.Debugf("New %s with id %s created", lyncserRootName, d.lyncserRootId)
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
			if !strings.HasPrefix(path, "~/") {
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
	fileId, _ := d.getFiledId(path)
	return downloadFileContents(d.service, fileId)
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
	fileId, exists := d.getFiledId(path)
	if !exists {
		d.createFile(path, reader)
		return nil
	}
	driveFile := d.mapIdToFile[fileId]
	_, err := updateFileContents(d.service, driveFile, fileId, reader)
	return err
}

func (d *DriveFileStore) DeleteFile(file string) error {
	fileId, exists := d.getFiledId(file)
	if !exists {
		return nil
	}
	return deleteFile(d.service, fileId)
}

func (d *DriveFileStore) DeleteAllFiles() error {
	return deleteFile(d.service, d.lyncserRootId)
}

func (d *DriveFileStore) FileExists(path string) (bool, error) {
	_, ok := d.getFiledId(path)
	return ok, nil
}

// getFiledId returns the Google Drive file id for the given path if it exists, otherwise it returns false for
// the second return value.
func (d *DriveFileStore) getFiledId(path string) (string, bool) {
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
	dirId, err = createDir(d.service, dirName, parentId)
	if err != nil {
		return "", err
	}
	d.Logger.Debugf("Directory '%s' successfully created", dirName)
	d.mapPathToFileId[dirName] = dirId
	return dirId, nil
}

func (d *DriveFileStore) createFile(path string, reader io.Reader) error {
	dirId, err := d.createDirIfNecessary(filepath.Dir(path))
	if err != nil {
		return err
	}
	baseName := filepath.Base(path)
	driveFile, err := createFile(d.service, baseName, "text/plain", reader, dirId)
	if err != nil {
		return err
	}
	d.mapPathToFileId[path] = driveFile.Id
	d.mapIdToFile[driveFile.Id] = driveFile
	return nil
}