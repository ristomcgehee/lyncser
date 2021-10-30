package sync

import (
	"fmt"
	"os"
	"testing"
	time "time"

	"github.com/chrismcgehee/lyncser/utils"
	"github.com/go-bdd/gobdd"
	"github.com/golang/mock/gomock"
	"github.com/pasdam/mockit/mockit"
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

func fileExistsLocally(t gobdd.StepTest, ctx gobdd.Context) {
	fileInfoIf, _ := ctx.Get("fileInfo")
	fileInfo := fileInfoIf.(*MockFileInfo)
	syncedFileIf, _ := ctx.Get("syncedFile")
	syncedFile := syncedFileIf.(utils.SyncedFile)
	tTestingIf, _ := ctx.Get("testing.T")
	tTesting := tTestingIf.(*testing.T)
	m := mockit.MockFunc(tTesting, os.Stat)
	m.With(syncedFile.RealPath).Return(fileInfo, nil)
}

func cloudModifiedTime(t gobdd.StepTest, ctx gobdd.Context, modifiedTime string) {
	fileStoreIf, _ := ctx.Get("fileStore")
	fileStore := fileStoreIf.(*MockFileStore)
	syncedFileIf, _ := ctx.Get("syncedFile")
	syncedFile := syncedFileIf.(utils.SyncedFile)
	fileStore.EXPECT().
		GetCloudModifiedTime(gomock.Eq(syncedFile)).
		Return(convertTime(modifiedTime))
}

func localModifiedTime(t gobdd.StepTest, ctx gobdd.Context, modifiedTime string) {
	fileInfoIf, _ := ctx.Get("fileInfo")
	fileInfo := fileInfoIf.(*MockFileInfo)
	fileInfo.EXPECT().
		ModTime().
		Return(convertTime(modifiedTime))
}

func lastCloudUpdate(t gobdd.StepTest, ctx gobdd.Context, modifiedTime string) {
	stateDataIf, _ := ctx.Get("stateData")
	stateData := stateDataIf.(*StateData)
	syncedFileIf, _ := ctx.Get("syncedFile")
	syncedFile := syncedFileIf.(utils.SyncedFile)
	stateData.FileStateData[syncedFile.FriendlyPath] = &FileStateData{
		LastCloudUpdate: convertTime(modifiedTime),
	} 
}

func fileUpdatedCloud(t gobdd.StepTest, ctx gobdd.Context) {
	fileStoreIf, _ := ctx.Get("fileStore")
	fileStore := fileStoreIf.(*MockFileStore)
	syncedFileIf, _ := ctx.Get("syncedFile")
	syncedFile := syncedFileIf.(utils.SyncedFile)
	fileStore.EXPECT().
		UpdateFile(gomock.Eq(syncedFile))
}

func convertTime(timeStr string) time.Time {
	timeToUse := ""
	switch timeStr {
	case "7 am":
		timeToUse = "2021-10-01T07:00:00.000Z"
	case "8 am":
		timeToUse = "2021-10-01T08:00:00.000Z"
	case "9 am":
		timeToUse = "2021-10-01T09:00:00.000Z"
	default:
		panic(fmt.Sprintf("unrecognized time: %s", timeToUse))
	}
	retTime, err := time.Parse(utils.TimeFormat, timeToUse)
	utils.PanicError(err)
	return retTime
}

func TestScenarios(t *testing.T) {
	fileName := "~/.bashrc"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	syncedFile := utils.SyncedFile{
		FriendlyPath: fileName,
		RealPath:     "/home/chris/.bashrc",
	}
	stateData := &StateData{
		FileStateData: map[string]*FileStateData{},
	}
	fileStore := NewMockFileStore(ctrl)
	fileInfo := NewMockFileInfo(ctrl)
	suite := gobdd.NewSuite(t, gobdd.WithBeforeScenario(func(ctx gobdd.Context) {
		ctx.Set("testing.T", t)
		ctx.Set("fileStore", fileStore)
		ctx.Set("fileInfo", fileInfo)
		ctx.Set("syncedFile", syncedFile)
		ctx.Set("stateData", stateData)
	}))
	suite.AddStep(`the file exists in the cloud`, fileExistsInCloud)
	suite.AddStep(`the cloud modified time is {text}`, cloudModifiedTime)
	suite.AddStep(`the file exists locally`, fileExistsLocally)
	suite.AddStep(`the local modified time is {text}`, localModifiedTime)
	suite.AddStep(`the last cloud update was {text}`, lastCloudUpdate)
	suite.AddStep(`the file should be updated to the cloud`, fileUpdatedCloud)
	suite.Run()
	handleFile(fileName, stateData, fileStore)
}
