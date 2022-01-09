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

type SyncedFile struct {
	FriendlyPath string
	RealPath     string
}

type Syncer struct {
	RemoteFileStore utils.FileStore
	LocalFileStore  utils.FileStore
	stateData       *LocalStateData
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
	s.stateData, err = getLocalStateData()
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
				if err = s.handleFile(path); err != nil {
					fmt.Printf("Error syncing file '%s': %v\n", path, err)
				}
				remoteFilesToHandle = utils.Remove(func(item string) bool {
					return item == path
				}, remoteFilesToHandle)
				return nil
			})
			if err != nil {
				fmt.Printf("Error walking dir '%s': %v\n", pathToSync, err)
			}

			// For any files that were not found locally, we'll download them now.
			for _, remoteFile := range remoteFilesToHandle {
				if err = s.handleFile(remoteFile); err != nil {
					fmt.Printf("Error syncing remote file '%s': %v\n", remoteFile, err)
				}
			}
		}
	}
	// globalConfigPath gets uploaded even if it's not explicitly listed
	// if err = s.handleFile(globalConfigPath); err != nil {
	// 	fmt.Printf("Error syncing file '%s': %v\n", globalConfigPath, err)
	// }

	if _, err = s.cleanupRemoteFiles(remoteFiles, globalConfig); err != nil {
		return err
	}

	return saveLocalStateData(s.stateData)
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
	file := SyncedFile{
		FriendlyPath: fileName,
		RealPath:     realPath,
	}
	fileExistsLocally, err := s.LocalFileStore.FileExists(file.RealPath)
	if err != nil {
		return err
	}
	if _, ok := s.stateData.FileStateData[fileName]; !ok {
		s.stateData.FileStateData[fileName] = &LocalFileStateData{
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
	fileExistsRemotely, err := s.RemoteFileStore.FileExists(file.FriendlyPath)
	if err != nil {
		return err
	}
	if fileExistsRemotely {
		if err = s.syncExistingFile(file, fileExistsLocally); err != nil {
			return err
		}
	} else {
		if !fileExistsLocally {
			// The file doesn't exist locally or in the cloud. ¯\_(ツ)_/¯
			return nil
		}
		if err = s.uploadFile(file); err != nil {
			return err
		}
		fmt.Printf("File '%s' successfully uploaded\n", file.FriendlyPath)
	}
	s.stateData.FileStateData[file.FriendlyPath].LastCloudUpdate = time.Now().UTC()
	return nil
}

// syncExistingFile uploads/downloads the file as necessary
func (s *Syncer) syncExistingFile(file SyncedFile, fileExistsLocally bool) error {
	modTimeCloud, err := s.RemoteFileStore.GetModifiedTime(file.FriendlyPath)
	if err != nil {
		return err
	}
	var modTimeLocal time.Time
	if fileExistsLocally {
		modTimeLocal, err = s.LocalFileStore.GetModifiedTime(file.RealPath)
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
		if err = s.uploadFile(file); err != nil {
			return err
		}
		fmt.Printf("File '%s' successfully uploaded\n", file.FriendlyPath)
	} else if downloadFile {
		if err = s.downloadFile(file); err != nil {
			return err
		}
		fmt.Printf("File '%s' successfully downloaded\n", file.FriendlyPath)
	} else if markDeleted {
		// mark the file as deleted so it's not downloaded again
		s.stateData.FileStateData[file.FriendlyPath].DeletedLocal = true
	}
	return nil
}

func (s *Syncer) uploadFile(file SyncedFile) error {
	contentReader, err := s.LocalFileStore.GetFileContents(file.RealPath)
	if err != nil {
		return err
	}
	defer contentReader.Close()
	err = s.RemoteFileStore.WriteFileContents(file.FriendlyPath, contentReader)
	if err != nil {
		return err
	}
	return nil
}

func (s *Syncer) downloadFile(file SyncedFile) error {
	contentReader, err := s.RemoteFileStore.GetFileContents(file.FriendlyPath)
	if err != nil {
		return err
	}
	defer contentReader.Close()
	err = s.LocalFileStore.WriteFileContents(file.RealPath, contentReader)
	if err != nil {
		return err
	}
	return nil
}

func (s *Syncer) cleanupRemoteFiles(remoteFiles []utils.StoredFile, globalConfig *GlobalConfig) (*RemoteStateData, error) {
	remoteStateData, err := getRemoteStateData(s.RemoteFileStore)
	if err != nil {
		return remoteStateData, err
	}

	for _, remoteFile := range remoteFiles {
		if strings.HasPrefix(globalConfigPath, remoteFile.Path) {
			continue // Because globalConfigPath is not in globalConfig.TagPaths, we need to skip it here.
		}
		inGlobalConfig := false
		for _, filesToSyncForTag := range globalConfig.TagPaths {
			for _, fileToSync := range filesToSyncForTag {
				if strings.HasPrefix(remoteFile.Path, fileToSync) || strings.HasPrefix(fileToSync, remoteFile.Path) {
					inGlobalConfig = true
				}
			}
		}
		if inGlobalConfig {
			delete(remoteStateData.FileStateData, remoteFile.Path)
		} else {
			_, exists := remoteStateData.FileStateData[remoteFile.Path]
			if !exists {
				remoteStateData.FileStateData[remoteFile.Path] = &RemoteFileStateData{
					MarkDeleted: time.Now(),
				}
			}
		}
	}

	// Delete files remotely if marked deleted more than 30 days ago.
	for filePath, fileData := range remoteStateData.FileStateData {
		if fileData.MarkDeleted.After(time.Now().AddDate(0, 0, -30)) {
			continue
		}
		if err = s.RemoteFileStore.DeleteFile(filePath); err != nil {
			return remoteStateData, err
		}
		delete(remoteStateData.FileStateData, filePath)
		fmt.Printf("File '%s' deleted remotely\n", filePath)
	}

	if err := saveRemoteStateData(remoteStateData, s.RemoteFileStore); err != nil {
		return remoteStateData, err
	}

	return remoteStateData, nil
}
