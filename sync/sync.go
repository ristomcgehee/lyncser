package sync

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chrismcgehee/lyncser/filestore"
	"github.com/chrismcgehee/lyncser/utils"
)

type SyncedFile struct {
	FriendlyPath string
	RealPath     string
	IsRemoteDir  bool
}

type HandleFileOutcome int

const (
	DownloadedFile HandleFileOutcome = iota
	UploadedFile
	MarkedDeleted
	NoChange
)

type Syncer struct {
	RemoteFileStore filestore.FileStore
	LocalFileStore  filestore.FileStore
	Logger          utils.Logger
	// Used to encrypt files stored in the remote file store.
	Encryptor utils.ReaderEncryptor
	// ForceDownload will download a file even if the local modified time is after the remote modified time.
	ForceDownload bool
	stateData     *LocalStateData
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
			if err := s.syncPath(pathToSync, remoteFiles); err != nil {
				s.Logger.Errorf("Error syncing path '%s': %s", pathToSync, err)
			}
		}
	}
	// globalConfigPath gets uploaded even if it's not explicitly listed
	handleFileOutcome, err := s.handleFile(globalConfigPath, false)
	if err != nil {
		s.Logger.Errorf("Error syncing file '%s': %v", globalConfigPath, err)
	}
	if handleFileOutcome == DownloadedFile {
		err = s.PerformSync()
		if err != nil {
			return err
		}
	}
	if _, err = s.cleanupRemoteFiles(remoteFiles, globalConfig); err != nil {
		return err
	}

	return saveLocalStateData(s.stateData)
}

// syncPath syncs the given path, recursively if it's a directory.
func (s *Syncer) syncPath(pathToSync string, remoteFiles []*filestore.StoredFile) error {
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
		var remoteFile *filestore.StoredFile
		idxRemoteFile := -1
		for i, remoteFileToHandle := range remoteFilesToHandle {
			if remoteFileToHandle.Path == path {
				remoteFile = remoteFileToHandle
				idxRemoteFile = i
				break
			}
		}
		isRemoteDir := remoteFile != nil && remoteFile.IsDir
		if _, err = s.handleFile(path, isRemoteDir); err != nil {
			s.Logger.Errorf("Error syncing file '%s': %v", path, err)
		}
		if remoteFile != nil {
			remoteFilesToHandle = append(remoteFilesToHandle[:idxRemoteFile], remoteFilesToHandle[idxRemoteFile+1:]...)
		}
		return nil
	})
	if err != nil {
		s.Logger.Errorf("Error walking directory '%s': %v", pathToSync, err)
	}

	// For any files that were not found locally, we'll download them now.
	for _, remoteFile := range remoteFilesToHandle {
		if _, err = s.handleFile(remoteFile.Path, remoteFile.IsDir); err != nil {
			s.Logger.Errorf("Error syncing remote file '%s': %v", remoteFile, err)
		}
	}

	return nil
}

// Get all the remote files that start with pathToSync if it is a directory.
func getMatchingRemoteFiles(pathToSync, realPath string, remoteFiles []*filestore.StoredFile) []*filestore.StoredFile {
	remoteFilesToHandle := make([]*filestore.StoredFile, 0)
	//nolint:errcheck
	stat, _ := os.Stat(realPath)
	if stat != nil && !stat.IsDir() {
		return remoteFilesToHandle
	}
	for _, remoteFile := range remoteFiles {
		if !strings.HasPrefix(remoteFile.Path, pathToSync) {
			continue
		}
		remoteFilesToHandle = append(remoteFilesToHandle, remoteFile)
	}
	return remoteFilesToHandle
}

// Creates the file if it does not exist in the cloud, otherwise downloads or uploads the file to the cloud.
func (s *Syncer) handleFile(fileName string, isRemoteDir bool) (HandleFileOutcome, error) {
	realPath, err := utils.RealPath(fileName)
	if err != nil {
		return NoChange, err
	}
	file := SyncedFile{
		FriendlyPath: fileName,
		RealPath:     realPath,
		IsRemoteDir:  isRemoteDir,
	}
	fileExistsLocally, err := s.LocalFileStore.FileExists(file.RealPath)
	if err != nil {
		return NoChange, err
	}
	if _, ok := s.stateData.FileStateData[file.FriendlyPath]; !ok {
		s.stateData.FileStateData[file.FriendlyPath] = &LocalFileStateData{
			LastCloudUpdate: utils.GetNeverSynced(),
		}
	}
	// Once a file is deleted locally, it's not downloaded again.
	if !fileExistsLocally && s.stateData.FileStateData[file.FriendlyPath].DeletedLocal {
		return NoChange, nil
	}
	if fileExistsLocally {
		s.stateData.FileStateData[file.FriendlyPath].DeletedLocal = false
	}
	s.Logger.Infof("Syncing %s", file.FriendlyPath)
	fileExistsRemotely, err := s.RemoteFileStore.FileExists(file.FriendlyPath)
	if err != nil {
		return NoChange, err
	}
	if !fileExistsRemotely && !fileExistsLocally {
		s.Logger.Warnf("File '%s' does not exist locally or remotely", file.FriendlyPath) // ¯\_(ツ)_/¯
		return NoChange, nil
	}
	handleFileOutcome, err := s.syncFile(file, fileExistsLocally, fileExistsRemotely)
	if err != nil {
		return handleFileOutcome, err
	}
	s.stateData.FileStateData[file.FriendlyPath].LastCloudUpdate = time.Now().UTC()
	return handleFileOutcome, nil
}

// Returns true if the file should be uploaded.
func doUploadFile(fileExistsLocally, fileExistsRemotely bool, modTimeLocal, modTimeCloud,
	lastCloudUpdate time.Time) bool {
	if !fileExistsRemotely {
		return true
	}
	if !fileExistsLocally {
		return false
	}
	return modTimeLocal.After(modTimeCloud) && utils.HasBeenSynced(lastCloudUpdate) &&
		modTimeLocal.After(lastCloudUpdate)
}

// Returns true if the file should be downloaded.
func doDownloadFile(fileExistsLocally, isRemoteDir, forceDownload bool, modTimeLocal, modTimeCloud,
	lastCloudUpdate time.Time) bool {
	if isRemoteDir {
		return false
	}
	if forceDownload {
		return true
	}
	if !fileExistsLocally && !utils.HasBeenSynced(lastCloudUpdate) {
		return true
	}
	return fileExistsLocally && modTimeCloud.After(modTimeLocal) && lastCloudUpdate.Before(modTimeCloud)
}

// Returns true if the files should be marked as deleted.
func doMarkDeleted(fileExistsLocally bool, lastCloudUpdate time.Time) bool {
	return !fileExistsLocally && utils.HasBeenSynced(lastCloudUpdate)
}

// syncFile uploads/downloads the file as necessary.
func (s *Syncer) syncFile(file SyncedFile, fileExistsLocally, fileExistsRemotely bool) (HandleFileOutcome, error) {
	var err error
	var modTimeCloud time.Time
	if fileExistsRemotely {
		modTimeCloud, err = s.RemoteFileStore.GetModifiedTime(file.FriendlyPath)
		if err != nil {
			return NoChange, err
		}
	}
	var modTimeLocal time.Time
	if fileExistsLocally {
		modTimeLocal, err = s.LocalFileStore.GetModifiedTime(file.RealPath)
		if err != nil {
			return NoChange, err
		}
		modTimeLocal = modTimeLocal.UTC()
	}
	lastCloudUpdate := s.stateData.FileStateData[file.FriendlyPath].LastCloudUpdate

	downloadFile := doDownloadFile(fileExistsLocally, file.IsRemoteDir, s.ForceDownload, modTimeLocal, modTimeCloud,
		lastCloudUpdate)
	uploadFile := doUploadFile(fileExistsLocally, fileExistsRemotely, modTimeLocal, modTimeCloud, lastCloudUpdate)
	markDeleted := doMarkDeleted(fileExistsLocally, lastCloudUpdate)

	switch {
	case downloadFile:
		if err := s.downloadFile(file); err != nil {
			return NoChange, err
		}
		s.Logger.Infof("File '%s' successfully downloaded", file.FriendlyPath)
		return DownloadedFile, nil
	case uploadFile:
		if err := s.uploadFile(file); err != nil {
			return NoChange, err
		}
		s.Logger.Infof("File '%s' successfully uploaded", file.FriendlyPath)
		return UploadedFile, nil
	case markDeleted:
		// mark the file as deleted so it's not downloaded again
		s.stateData.FileStateData[file.FriendlyPath].DeletedLocal = true
		return MarkedDeleted, nil
	}
	return NoChange, nil
}

func (s *Syncer) uploadFile(file SyncedFile) error {
	contentReader, err := s.LocalFileStore.GetFileContents(file.RealPath)
	if err != nil {
		return err
	}
	defer contentReader.Close()
	readerEncrypted, err := s.Encryptor.EncryptReader(contentReader)
	if err != nil {
		return err
	}
	err = s.RemoteFileStore.WriteFileContents(file.FriendlyPath, readerEncrypted)
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
	decryptedReader, err := s.Encryptor.DecryptReader(contentReader)
	if err != nil {
		return err
	}

	err = s.LocalFileStore.WriteFileContents(file.RealPath, decryptedReader)
	if err != nil {
		return err
	}
	return nil
}

func (s *Syncer) cleanupRemoteFiles(remoteFiles []*filestore.StoredFile,
	globalConfig *GlobalConfig) (*RemoteStateData, error) {
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
		if err := s.RemoteFileStore.DeleteFile(filePath); err != nil {
			return remoteStateData, err
		}
		delete(remoteStateData.FileStateData, filePath)
		s.Logger.Infof("File '%s' deleted remotely", filePath)
	}

	if err := saveRemoteStateData(remoteStateData, s.RemoteFileStore); err != nil {
		return remoteStateData, err
	}

	return remoteStateData, nil
}
