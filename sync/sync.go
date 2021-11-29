package sync

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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
func (s *Syncer) PerformSync() error {
	globalConfig, err := getGlobalConfig()
	if err != nil {
		return err
	}
	localConfig, err := getLocalConfig()
	if err != nil {
		return err
	}
	s.stateData, err = getStateData()
	if err != nil {
		return err
	}

	remoteFiles, err := s.RemoteFileStore.GetFiles()
	if err != nil {
		return err
	}

	for tag, paths := range globalConfig.TagPaths {
		if !utils.InSlice(tag, localConfig.Tags) {
			continue
		}
		for _, pathToSync := range paths {
			realPath, err := utils.RealPath(pathToSync)
			if err != nil {
				return err
			}
			realPath, err = filepath.EvalSymlinks(realPath)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			remoteFilesToHandle := getMatchingRemoteFiles(pathToSync, realPath, remoteFiles)

			// Recursively sync pathToSync.
			err = filepath.WalkDir(realPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}
				if d != nil && (d.IsDir() || !d.Type().IsRegular()) {
					return nil
				}
				path = strings.Replace(path, realPath, pathToSync, 1)
				err = s.handleFile(path)
				if err != nil {
					return err
				}
				remoteFilesToHandle = utils.Remove(func(item string) bool {
					return item == path
				}, remoteFilesToHandle)
				return nil
			})
			if err != nil {
				return err
			}

			// For any files that were not found locally, we'll download them now.
			for _, remoteFile := range remoteFilesToHandle {
				err = s.handleFile(remoteFile)
				if err != nil {
					return err
				}
			}
		}
	}
	// globalConfigPath gets uploaded even if it's not explicitly listed
	err = s.handleFile(globalConfigPath)
	if err != nil {
		return err
	}

	err = saveStateData(s.stateData)
	return err
}

// Get all the remote files that start with pathToSync if it is a directory.
func getMatchingRemoteFiles(pathToSync, realPath string, remoteFiles []utils.StoredFile) []string {
	remoteFilesToHandle := make([]string, 0)
	stat, _ := os.Stat(realPath)
	if stat != nil && !stat.IsDir() {
		return remoteFilesToHandle
	}
	for _, remoteFile := range remoteFiles {
		if remoteFile.IsDir || !strings.HasPrefix(remoteFile.Path, pathToSync) {
			continue
		}
		remoteFilesToHandle = append(remoteFilesToHandle, remoteFile.Path)
	}
	return remoteFilesToHandle
}

// Creates the file if it does not exist in the cloud, otherwise downloads or uploads the file to the cloud
func (s *Syncer) handleFile(fileName string) error {
	realPath, err := utils.RealPath(fileName)
	if err != nil {
		return err
	}
	file := utils.SyncedFile{
		FriendlyPath: fileName,
		RealPath:     realPath,
	}
	fileExistsLocally, err := s.LocalFileStore.FileExists(file)
	if err != nil {
		return err
	}
	if _, ok := s.stateData.FileStateData[fileName]; !ok {
		s.stateData.FileStateData[fileName] = &FileStateData{
			LastCloudUpdate: utils.GetNeverSynced(),
		}
	}
	// Once a file is deleted locally, it's not downloaded again.
	if !fileExistsLocally && s.stateData.FileStateData[file.FriendlyPath].DeletedLocal {
		return nil
	}
	if fileExistsLocally {
		s.stateData.FileStateData[file.FriendlyPath].DeletedLocal = false
	}
	fmt.Println("Syncing", fileName)
	fileExistsRemotely, err := s.RemoteFileStore.FileExists(file)
	if err != nil {
		return err
	}
	if fileExistsRemotely {
		err = s.syncExistingFile(file, fileExistsLocally)
		if err != nil {
			return err
		}
	} else {
		if !fileExistsLocally {
			// The file doesn't exist locally or in the cloud. ¯\_(ツ)_/¯
			return nil
		}
		err = s.RemoteFileStore.CreateFile(file)
		if err != nil {
			return err
		}
		fmt.Printf("File '%s' successfully created\n", file.FriendlyPath)
	}
	s.stateData.FileStateData[file.FriendlyPath].LastCloudUpdate = time.Now().UTC()
	return nil
}

// syncExistingFile uploads/downloads the file as necessary
func (s *Syncer) syncExistingFile(file utils.SyncedFile, fileExistsLocally bool) error {
	modTimeCloud, err := s.RemoteFileStore.GetModifiedTime(file)
	if err != nil {
		return err
	}
	var modTimeLocal time.Time
	if fileExistsLocally {
		modTimeLocal, err = s.LocalFileStore.GetModifiedTime(file)
		if err != nil {
			return err
		}
		modTimeLocal = modTimeLocal.UTC()
	}
	lastCloudUpdate := s.stateData.FileStateData[file.FriendlyPath].LastCloudUpdate

	uploadFile := fileExistsLocally && modTimeLocal.After(modTimeCloud) &&
		utils.HasBeenSynced(lastCloudUpdate) && modTimeLocal.After(lastCloudUpdate)
	downloadFile := (!fileExistsLocally && !utils.HasBeenSynced(lastCloudUpdate)) ||
		(fileExistsLocally && modTimeCloud.After(modTimeLocal) && lastCloudUpdate.Before(modTimeCloud))
	markDeleted := !fileExistsLocally && utils.HasBeenSynced(lastCloudUpdate)

	if uploadFile {
		err = s.RemoteFileStore.UpdateFile(file)
		if err != nil {
			return err
		}
		fmt.Printf("File '%s' successfully uploaded\n", file.FriendlyPath)
	} else if downloadFile {
		err = s.RemoteFileStore.DownloadFile(file)
		if err != nil {
			return err
		}
		fmt.Printf("File '%s' successfully downloaded\n", file.FriendlyPath)
	} else if markDeleted {
		// mark the file as deleted so it's not downloaded again
		s.stateData.FileStateData[file.FriendlyPath].DeletedLocal = true
	}
	return nil
}
