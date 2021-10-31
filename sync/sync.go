package sync

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/chrismcgehee/lyncser/utils"
)

type Syncer struct {
	RemoteFileStore utils.FileStore
	LocalFileStore  utils.FileStore
	stateData       *StateData
}

// PerformSync does the entire sync from end to end.
func (s *Syncer) PerformSync() {
	globalConfig := getGlobalConfig()
	localConfig := getLocalConfig()
	s.stateData = getStateData()

	remoteFiles := s.RemoteFileStore.GetFiles()

	for tag, paths := range globalConfig.TagPaths {
		if !utils.InSlice(tag, localConfig.Tags) {
			continue
		}
		for _, pathToSync := range paths {
			// Get all the remote files that start with pathToSync.
			remoteFilesToHandle := make([]string, 0)
			for _, remoteFile := range remoteFiles {
				if remoteFile.IsDir || !strings.HasPrefix(remoteFile.Path, pathToSync) {
					continue
				}
				remoteFilesToHandle = append(remoteFilesToHandle, remoteFile.Path)
			}

			// Recursively sync pathToSync. 
			realpath, err := filepath.EvalSymlinks(utils.RealPath(pathToSync))
			utils.PanicError(err)
			filepath.WalkDir(realpath, func(path string, d fs.DirEntry, err error) error {
				var pathError *fs.PathError
				if errors.As(err, &pathError) && pathError.Err.Error() != "no such file or directory" {
					utils.PanicError(err)
				}
				if d != nil && (d.IsDir() || !d.Type().IsRegular()) {
					return nil
				}
				path = strings.Replace(path, realpath, pathToSync, 1)
				s.handleFile(path)
				remoteFilesToHandle = utils.Remove(func(item string) bool {
					return utils.RealPath(item) == path
				}, remoteFilesToHandle)
				return nil
			})

			// For any files that were not found locally, we'll download them now.
			for _, remoteFile := range remoteFilesToHandle {
				s.handleFile(remoteFile)
			}
		}
	}
	// globalConfigPath gets uploaded even if it's not explicitly listed
	s.handleFile(globalConfigPath)

	saveStateData(s.stateData)
}

// Creates the file if it does not exist in the cloud, otherwise downloads or uploads the file to the cloud
func (s *Syncer) handleFile(fileName string) {
	fmt.Println("Syncing", fileName)
	file := utils.SyncedFile{
		FriendlyPath: fileName,
		RealPath:     utils.RealPath(fileName),
	}
	fileExistsLocally := s.LocalFileStore.FileExists(file)
	if _, ok := s.stateData.FileStateData[fileName]; !ok {
		neverUpdated, _ := time.Parse(utils.TimeFormat, "2000-01-01T01:01:01.000Z")
		s.stateData.FileStateData[fileName] = &FileStateData{
			LastCloudUpdate: neverUpdated,
		}
	}
	if s.RemoteFileStore.FileExists(file) {
		s.syncExistingFile(file, fileExistsLocally)
	} else {
		if !fileExistsLocally {
			return
		}
		s.RemoteFileStore.CreateFile(file)
		fmt.Printf("File '%s' successfully created\n", file.FriendlyPath)
	}
	s.stateData.FileStateData[file.FriendlyPath].LastCloudUpdate = time.Now().UTC()
}

// syncExistingFile uploads/downloads the file as necessary
func (s *Syncer) syncExistingFile(file utils.SyncedFile, fileExistsLocally bool) {
	modTimeCloud := s.RemoteFileStore.GetModifiedTime(file)
	var modTimeLocal time.Time
	if fileExistsLocally {
		modTimeLocal = s.LocalFileStore.GetModifiedTime(file).UTC()
	}
	lastCloudUpdate := s.stateData.FileStateData[file.FriendlyPath].LastCloudUpdate

	if fileExistsLocally && modTimeLocal.After(modTimeCloud) && modTimeLocal.After(lastCloudUpdate) && lastCloudUpdate.Year() > 2001 {
		s.RemoteFileStore.UpdateFile(file)
		fmt.Printf("File '%s' successfully uploaded\n", file.FriendlyPath)
	} else if !fileExistsLocally || modTimeCloud.After(lastCloudUpdate) {
		s.RemoteFileStore.DownloadFile(file)
		fmt.Printf("File '%s' successfully downloaded\n", file.FriendlyPath)
	}
}
