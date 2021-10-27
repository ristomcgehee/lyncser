package sync

import (
	// "os"
	"testing"
	// time "time"

	"github.com/chrismcgehee/lyncser/utils"
	"github.com/go-bdd/gobdd"
	"github.com/golang/mock/gomock"
	// "github.com/pasdam/mockit/mockit"
)

func fileExistsInCloud(t gobdd.StepTest, ctx gobdd.Context) {
	fileStoreIf, _ := ctx.Get("fileStore")
	fileStore := fileStoreIf.(*MockFileStore)
	syncedFileIf, _ := ctx.Get("syncedFile")
	syncedFile := syncedFileIf.(utils.SyncedFile)
	fileStore.EXPECT().
		FileExistsCloud(gomock.Eq(syncedFile)).
		Return(true)
}

func TestScenarios(t *testing.T) {
	fileName := "~/.bashrc"
	const oct1st7am = "2021-10-01T07:00:00.000Z"
	const oct1st8am = "2021-10-01T08:00:00.000Z"
	const oct1st9am = "2021-10-01T09:00:00.000Z"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	syncedFile := utils.SyncedFile{
		FriendlyPath: fileName,
		RealPath:     "/home/chris/.bashrc",
	}
	fileStore := NewMockFileStore(ctrl)
	suite := gobdd.NewSuite(t, gobdd.WithBeforeScenario(func(ctx gobdd.Context) {
		ctx.Set("ctrl", ctrl)
		ctx.Set("fileStore", fileStore)
		ctx.Set("syncedFile", syncedFile)
	}))
	suite.AddStep(`the file exists in the cloud`, fileExistsInCloud)
	// suite.AddStep(`I the result should equal (\d+)`, check)
	suite.Run()
	stateData := &StateData{
		FileStateData: map[string]*FileStateData{
			fileName: {
				LastCloudUpdate: oct1st8am,
			},
		},
	}
	handleFile(fileName, stateData, fileStore)
}

// func TestSnyc(t *testing.T) {
// 	t.Parallel()

// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	time7am, err := time.Parse(utils.TimeFormat, oct1st7am)
// 	utils.PanicError(err)
// 	fileStore.EXPECT().
// 		GetCloudModifiedTime(gomock.Eq(syncedFile)).
// 		Return(time7am)
// 	fileInfo := NewMockFileInfo(ctrl)
// 	time9am, err := time.Parse(utils.TimeFormat, oct1st9am)
// 	utils.PanicError(err)
// 	fileInfo.EXPECT().
// 		ModTime().
// 		Return(time9am)
// 	m := mockit.MockFunc(t, os.Stat)
// 	m.With(syncedFile.RealPath).Return(fileInfo, nil)
// 	handleFile(fileName, stateData, fileStore)
// }
