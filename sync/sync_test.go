package sync

import (
	"fmt"
	"testing"
	time "time"

	"github.com/go-bdd/gobdd"
	"github.com/golang/mock/gomock"

	"github.com/chrismcgehee/lyncser/utils"
)

func panicError(err error) {
	if err != nil {
		panic(err)
	}
}

// Gets the context used by most steps.
func unwrapContext(ctx gobdd.Context) (*Syncer, utils.SyncedFile) {
	syncerIf, err := ctx.Get("syncer")
	panicError(err)
	syncer := syncerIf.(*Syncer)
	syncedFileIf, err := ctx.Get("syncedFile")
	panicError(err)
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
	panicError(err)
	return retTime
}

// List of functions that will be called after the scenario has finished.
type assertExpectationFunc func(t gobdd.StepTest, ctx gobdd.Context)

// Local file info =================================================================================

func fileExistsLocally(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	localFileStore := syncer.LocalFileStore.(*MockFileStore)
	localFileStore.EXPECT().
		FileExists(gomock.Eq(syncedFile)).
		Return(true, nil).AnyTimes()
}

func fileDoesntExistLocally(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	localFileStore := syncer.LocalFileStore.(*MockFileStore)
	localFileStore.EXPECT().
		FileExists(gomock.Eq(syncedFile)).
		Return(false, nil).AnyTimes()
}

func localModifiedTime(t gobdd.StepTest, ctx gobdd.Context, modifiedTime string) {
	syncer, syncedFile := unwrapContext(ctx)
	localFileStore := syncer.LocalFileStore.(*MockFileStore)
	localFileStore.EXPECT().
		GetModifiedTime(gomock.Eq(syncedFile)).
		Return(convertTime(modifiedTime), nil).AnyTimes()
}

func lastCloudUpdate(t gobdd.StepTest, ctx gobdd.Context, modifiedTime string) {
	syncer, syncedFile := unwrapContext(ctx)
	syncer.stateData.FileStateData[syncedFile.FriendlyPath].LastCloudUpdate = convertTime(modifiedTime)
}

func wasMarkedDeletedLocally(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	syncer.stateData.FileStateData[syncedFile.FriendlyPath].DeletedLocal = true
}

// Remote file info ================================================================================

func fileExistsInCloud(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		FileExists(gomock.Eq(syncedFile)).
		Return(true, nil).AnyTimes()
}

func fileDoesntExistInCloud(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		FileExists(gomock.Eq(syncedFile)).
		Return(false, nil).AnyTimes()
}

func cloudModifiedTime(t gobdd.StepTest, ctx gobdd.Context, modifiedTime string) {
	syncer, syncedFile := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		GetModifiedTime(gomock.Eq(syncedFile)).
		Return(convertTime(modifiedTime), nil).AnyTimes()
}

// Actions =========================================================================================

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

func shouldBeDeletedLocally(t gobdd.StepTest, ctx gobdd.Context) {
	iface, err := ctx.Get("expectations")
	panicError(err)
	expectations := iface.([]assertExpectationFunc)
	expectations = append(expectations, func(t gobdd.StepTest, ctx gobdd.Context) {
		syncer, syncedFile := unwrapContext(ctx)
		if !syncer.stateData.FileStateData[syncedFile.FriendlyPath].DeletedLocal {
			t.Fail()
		}
	})
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
		syncer.stateData.FileStateData[syncedFile.FriendlyPath] = &FileStateData{}
		ctx.Set("syncer", syncer)
		ctx.Set("syncedFile", syncedFile)
		expectations := []assertExpectationFunc{}
		ctx.Set("expectations", expectations)
	}), gobdd.WithAfterScenario(func(ctx gobdd.Context) {
		syncer.handleFile(syncedFile.FriendlyPath)
		iface, _ := ctx.Get("expectations")
		expectations := iface.([]assertExpectationFunc)
		for _, assertExpectation := range expectations {
			assertExpectation(t, ctx)
		}
	}))
	suite.AddParameterTypes(`{text}`, []string{`"([\d\w\-\:\s]+)"`})

	// local file
	suite.AddStep(`the file exists locally`, fileExistsLocally)
	suite.AddStep(`the file does not exist locally`, fileDoesntExistLocally)
	suite.AddStep(`the local modified time is {text}`, localModifiedTime)
	suite.AddStep(`the last cloud update was {text}`, lastCloudUpdate)
	suite.AddStep(`the file was marked deleted locally`, wasMarkedDeletedLocally)
	// cloud file
	suite.AddStep(`the file exists in the cloud`, fileExistsInCloud)
	suite.AddStep(`the file does not exist in the cloud`, fileDoesntExistInCloud)
	suite.AddStep(`the cloud modified time is {text}`, cloudModifiedTime)
	// actions
	suite.AddStep(`the file should be updated to the cloud`, fileUpdatedCloud)
	suite.AddStep(`the file should be created in the cloud`, fileCreatedCloud)
	suite.AddStep(`the file should be downloaded from the cloud`, fileDownloadedFromCloud)
	suite.AddStep(`the file should be marked deleted locally`, shouldBeDeletedLocally)
	suite.AddStep(`nothing should happen`, nothing)

	suite.Run()
}
