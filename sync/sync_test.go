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
	case "9:01 am":
		timeToUse = "2021-10-01T09:01:00.000Z"
	case "never":
		timeToUse = "2000-01-01T00:00:00.000Z"
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
		Return(true).AnyTimes()
}

func fileDoesntExistInCloud(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		FileExists(gomock.Eq(syncedFile)).
		Return(false).AnyTimes()
}

func fileExistsLocally(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	localFileStore := syncer.LocalFileStore.(*MockFileStore)
	localFileStore.EXPECT().
		FileExists(gomock.Eq(syncedFile)).
		Return(true).AnyTimes()
}

func fileDoesntExistLocally(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	localFileStore := syncer.LocalFileStore.(*MockFileStore)
	localFileStore.EXPECT().
		FileExists(gomock.Eq(syncedFile)).
		Return(false).AnyTimes()
}

func cloudModifiedTime(t gobdd.StepTest, ctx gobdd.Context, modifiedTime string) {
	syncer, syncedFile := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		GetModifiedTime(gomock.Eq(syncedFile)).
		Return(convertTime(modifiedTime)).AnyTimes()
}

func localModifiedTime(t gobdd.StepTest, ctx gobdd.Context, modifiedTime string) {
	syncer, syncedFile := unwrapContext(ctx)
	localFileStore := syncer.LocalFileStore.(*MockFileStore)
	localFileStore.EXPECT().
		GetModifiedTime(gomock.Eq(syncedFile)).
		Return(convertTime(modifiedTime)).AnyTimes()
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

func fileCreatedCloud(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		CreateFile(gomock.Eq(syncedFile))
}

func fileDownloadedFromCloud(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		DownloadFile(gomock.Eq(syncedFile))
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
	suite.AddParameterTypes(`{text}`, []string{`"([\d\w\-\:\s]+)"`})

	// local file
	suite.AddStep(`the file exists locally`, fileExistsLocally)
	suite.AddStep(`the file does not exist locally`, fileDoesntExistLocally)
	suite.AddStep(`the local modified time is {text}`, localModifiedTime)
	suite.AddStep(`the last cloud update was {text}`, lastCloudUpdate)
	// cloud file
	suite.AddStep(`the file exists in the cloud`, fileExistsInCloud)
	suite.AddStep(`the file does not exist in the cloud`, fileDoesntExistInCloud)
	suite.AddStep(`the cloud modified time is {text}`, cloudModifiedTime)
	// actions
	suite.AddStep(`the file should be updated to the cloud`, fileUpdatedCloud)
	suite.AddStep(`the file should be created in the cloud`, fileCreatedCloud)
	suite.AddStep(`the file should be downloaded from the cloud`, fileDownloadedFromCloud)
	// other
	suite.AddStep(`nothing should happen`, nothing)

	suite.Run()
}
