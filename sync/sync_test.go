package sync

import (
	"testing"

	"github.com/chrismcgehee/lyncser/utils"
	"github.com/golang/mock/gomock"
	"github.com/pasdam/mockit/mockit"
)

func TestSnyc(t *testing.T) {
	t.Parallel()
	m := mockit.MockFunc(t, getGlobalConfig)
	m.With().Return(GlobalConfig{
		TagPaths: map[string][]string{
			"all": {
				"~/.bashrc",
			},
		},
	})
	m = mockit.MockFunc(t, getLocalConfig)
	m.With().Return(LocalConfig{
		Tags: []string{
			"all",
		},
	})

	ctrl := gomock.NewController(t)
 	defer ctrl.Finish()

	syncedFile := utils.SyncedFile{
		FriendlyPath: "~/.bashrc",
		RealPath: "/home/chris/.bashrc",
	}
	fileStore := NewMockFileStore(ctrl)
	// fileStore.EXPECT().Initialize()
	fileStore.EXPECT().
		FileExistsCloud(gomock.Eq(syncedFile)).
		Return(true)
	stateData := &StateData{
		FileStateData: map[string]*FileStateData{
			"~/.bashrc": {
				LastCloudUpdate: "2021-10-02T17:04:48.526Z",
			},
		},
	}
	handleFile("~/.bashrc", stateData, fileStore)
}
