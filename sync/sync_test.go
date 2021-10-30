package sync

import (
	"fmt"
	"testing"
	time "time"

	"github.com/go-bdd/gobdd"
	"github.com/golang/mock/gomock"

	"github.com/chrismcgehee/lyncser/utils"
)

func unwrapContext(ctx gobdd.Context) (*Syncer, utils.SyncedFile) {
	syncerIf, err := ctx.Get("syncer")
	utils.PanicError(err)
	syncer := syncerIf.(*Syncer)
	syncedFileIf, err := ctx.Get("syncedFile")
	utils.PanicError(err)
	syncedFile := syncedFileIf.(utils.SyncedFile)
	return syncer, syncedFile
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

func fileExistsInCloud(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		FileExists(gomock.Eq(syncedFile)).
		Return(true)
}

func fileExistsLocally(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	localFileStore := syncer.LocalFileStore.(*MockFileStore)
	localFileStore.EXPECT().
		FileExists(gomock.Eq(syncedFile)).
		Return(true)
}

func cloudModifiedTime(t gobdd.StepTest, ctx gobdd.Context, modifiedTime string) {
	syncer, syncedFile := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		GetModifiedTime(gomock.Eq(syncedFile)).
		Return(convertTime(modifiedTime))
}

func localModifiedTime(t gobdd.StepTest, ctx gobdd.Context, modifiedTime string) {
	syncer, syncedFile := unwrapContext(ctx)
	localFileStore := syncer.LocalFileStore.(*MockFileStore)
	localFileStore.EXPECT().
		GetModifiedTime(gomock.Eq(syncedFile)).
		Return(convertTime(modifiedTime))
}

func lastCloudUpdate(t gobdd.StepTest, ctx gobdd.Context, modifiedTime string) {
	syncer, syncedFile := unwrapContext(ctx)
	syncer.stateData.FileStateData[syncedFile.FriendlyPath] = &FileStateData{
		LastCloudUpdate: convertTime(modifiedTime),
	}
}

func fileUpdatedCloud(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		UpdateFile(gomock.Eq(syncedFile))
}

func nothing(t gobdd.StepTest, ctx gobdd.Context) {
	// easy peasy
}

func TestScenarios(t *testing.T) {
	syncedFile := utils.SyncedFile{
		FriendlyPath: "~/test_file1",
		RealPath:     "/home/chris/test_file1",
	}
	syncer := &Syncer{
		stateData: &StateData{
			FileStateData: map[string]*FileStateData{},
		},
	}
	suite := gobdd.NewSuite(t, gobdd.WithBeforeScenario(func(ctx gobdd.Context) {
		ctrl := gomock.NewController(t)
		syncer.RemoteFileStore = NewMockFileStore(ctrl)
		syncer.LocalFileStore = NewMockFileStore(ctrl)
		ctx.Set("syncer", syncer)
		ctx.Set("syncedFile", syncedFile)
	}), gobdd.WithAfterScenario(func(ctx gobdd.Context) {
		syncer.handleFile(syncedFile.FriendlyPath)
	}))
	suite.AddStep(`the file exists in the cloud`, fileExistsInCloud)
	suite.AddStep(`the cloud modified time is {text}`, cloudModifiedTime)
	suite.AddStep(`the file exists locally`, fileExistsLocally)
	suite.AddStep(`the local modified time is {text}`, localModifiedTime)
	suite.AddStep(`the last cloud update was {text}`, lastCloudUpdate)
	suite.AddStep(`the file should be updated to the cloud`, fileUpdatedCloud)
	suite.AddStep(`nothing should happen`, nothing)
	suite.Run()
}
