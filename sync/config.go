package sync

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"
	time "time"

	yaml "gopkg.in/yaml.v3"

	"github.com/chrismcgehee/lyncser/filestore"
	"github.com/chrismcgehee/lyncser/utils"
)

const (
	// Holds state that helps determine whether a file should be uploaded or downloaded.
	stateLocalFilePath = "~/.config/lyncser/state.json"
	// Holds state that helps determine whether a file should be deleted remotely.
	stateRemoteFilePath = "~/.config/lyncser/stateRemote.json"
	// Contains global configuration used across all machines associated with this user.
	globalConfigPath = "~/.config/lyncser/globalConfig.yaml"
	// Contains configuration specific to this machine.
	localConfigPath = "~/.config/lyncser/localConfig.yaml"
	// Key for encrypting files.
	encryptionKeyPath = "~/.config/lyncser/encryption.key"
	// Length of encryption key.
	keyLengthBits = 256
)

type RemoteStateData struct {
	// Key is file path. Value is the state data associated with that file.
	FileStateData map[string]*RemoteFileStateData
}

type RemoteFileStateData struct {
	// The datetime this file was marked as deleted.
	MarkDeleted time.Time
}

type GlobalConfig struct {
	// Specifies which files should be synced for machines associated with each tag. The key in this map is the tag
	// name. The value is the list of files/directories that should be synced for that tag.
	TagPaths map[string][]string `yaml:"paths"`
}

type LocalConfig struct {
	// Specifies with tags this machine should be associated with.
	Tags []string `yaml:"tags"`
}

type LocalStateData struct {
	// Key is file path. Value is the state data associated with that file.
	FileStateData map[string]*LocalFileStateData
}

type LocalFileStateData struct {
	// The last time this file has been uploaded/downloaded from the cloud.
	LastCloudUpdate time.Time
	// Whether this file has been deleted locally.
	DeletedLocal bool
}

// getGlobalConfig reads and parses the global config file. If it does not exist, it return an empty config object.
func getGlobalConfig() (*GlobalConfig, error) {
	fullConfigPath, err := utils.RealPath(globalConfigPath)
	if err != nil {
		return nil, err
	}
	var config GlobalConfig
	data, err := ioutil.ReadFile(fullConfigPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		config = GlobalConfig{
			TagPaths: map[string][]string{},
		}
	case err != nil:
		return nil, err
	default:
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, err
		}
	}
	return &config, nil
}

// getLocalConfig reads and parses the local config file. If it does not exist, it will create it.
func getLocalConfig() (*LocalConfig, error) {
	fullConfigPath, err := utils.RealPath(localConfigPath)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(fullConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		configDir := path.Dir(fullConfigPath)
		if err := os.MkdirAll(configDir, 0o700); err != nil {
			return nil, err
		}
		data = []byte("tags:\n  - all\n")
		err = os.WriteFile(fullConfigPath, data, 0o644)
	}
	if err != nil {
		return nil, err
	}
	var config LocalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// getLocalStateData reads and parses the state data file. If that file does not exist yet, this method will return
// a newly initialized struct.
func getLocalStateData() (*LocalStateData, error) {
	var stateData LocalStateData
	realpath, err := utils.RealPath(stateLocalFilePath)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(realpath)
	if errors.Is(err, os.ErrNotExist) {
		stateData = LocalStateData{
			FileStateData: map[string]*LocalFileStateData{},
		}
	} else {
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &stateData); err != nil {
			return nil, err
		}
	}
	return &stateData, nil
}

// saveLocalStateData will save the state data to disk.
func saveLocalStateData(stateData *LocalStateData) error {
	data, err := json.MarshalIndent(stateData, "", " ")
	if err != nil {
		return err
	}
	realpath, err := utils.RealPath(stateLocalFilePath)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(realpath, data, 0o644)
}

// getRemoteStateData returns the state data that is stored remotely.
func getRemoteStateData(remoteFileStore filestore.FileStore) (*RemoteStateData, error) {
	exists, err := remoteFileStore.FileExists(stateRemoteFilePath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return &RemoteStateData{
			FileStateData: map[string]*RemoteFileStateData{},
		}, nil
	}

	contentsReader, err := remoteFileStore.GetFileContents(stateRemoteFilePath)
	if err != nil {
		return nil, err
	}
	defer contentsReader.Close()
	contents, err := ioutil.ReadAll(contentsReader)
	if err != nil {
		return nil, err
	}
	var stateData *RemoteStateData
	if err := json.Unmarshal(contents, &stateData); err != nil {
		return nil, err
	}
	return stateData, nil
}

func saveRemoteStateData(stateData *RemoteStateData, remoteFileStore filestore.FileStore) error {
	data, err := json.MarshalIndent(stateData, "", " ")
	if err != nil {
		return err
	}
	reader := bytes.NewReader(data)
	return remoteFileStore.WriteFileContents(stateRemoteFilePath, reader)
}

func GetEncryptionKey() ([]byte, error) {
	keyBytes := make([]byte, keyLengthBits/8)
	fullEncryptionKeyPath, err := utils.RealPath(encryptionKeyPath)
	if err != nil {
		return keyBytes, err
	}
	var keyHex string
	keyFileBytes, err := ioutil.ReadFile(fullEncryptionKeyPath)
	if errors.Is(err, os.ErrNotExist) {
		// Generate a new key.
		if _, err := rand.Read(keyBytes); err != nil {
			return keyBytes, err
		}
		keyHex = hex.EncodeToString(keyBytes)
		err = os.WriteFile(fullEncryptionKeyPath, []byte(keyHex), 0o600)
	} else if err == nil {
		keyHex = string(keyFileBytes)
	}
	if err != nil {
		return keyBytes, err
	}
	keyBytes, err = hex.DecodeString(strings.TrimSpace(keyHex))
	return keyBytes, err
}
