package sync

import (
	"fmt"
	"io"
	"strings"
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
	syncerIf, _ := ctx.Get("syncer")
	var syncer *Syncer
	if syncerIf != nil {
		syncer = syncerIf.(*Syncer)
	}
	syncedFileIf, _ := ctx.Get("syncedFile")
	var syncedFile utils.SyncedFile
	if syncedFileIf != nil {
		syncedFile = syncedFileIf.(utils.SyncedFile)
	}
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

// With gobdd the context set during steps cannot be accessed in the WithAfterScenario function, so that's
// why I'm using these global variables.
var expectations = []assertExpectationFunc{}
var globalConfig = &GlobalConfig{}
var remoteFiles = []utils.StoredFile{}

func addExpectation(t gobdd.StepTest, ctx gobdd.Context, expectation assertExpectationFunc) {
	expectations = append(expectations, expectation)
}

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

func globalConfigHasFile(t gobdd.StepTest, ctx gobdd.Context, filePath string) {
	globalConfig.TagPaths["all"] = append(globalConfig.TagPaths["all"], filePath)
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

func cloudHasFile(t gobdd.StepTest, ctx gobdd.Context, filePath string) {
	remoteFiles = append(remoteFiles, utils.StoredFile{
		Path: filePath,
	})
}

func remoteStateDataFileDoesNotExist(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, _ := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		FileExists(gomock.Eq(utils.SyncedFile{
			FriendlyPath: stateRemoteFileFile,
		})).
		Return(false, nil)
}

// Actions =========================================================================================

func fileUpdatedCloud(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	localFileStore := syncer.LocalFileStore.(*MockFileStore)
	localFileStore.EXPECT().
		GetFileContents(gomock.Eq(syncedFile)).
		Return(io.NopCloser(strings.NewReader("string")), nil)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		WriteFileContents(gomock.Eq(syncedFile), gomock.Any())
}

func fileDownloadedFromCloud(t gobdd.StepTest, ctx gobdd.Context) {
	syncer, syncedFile := unwrapContext(ctx)
	cloudFileStore := syncer.RemoteFileStore.(*MockFileStore)
	cloudFileStore.EXPECT().
		GetFileContents(gomock.Eq(syncedFile)).
		Return(io.NopCloser(strings.NewReader("string")), nil)
	localFileStore := syncer.LocalFileStore.(*MockFileStore)
	localFileStore.EXPECT().
		WriteFileContents(gomock.Eq(syncedFile), gomock.Any())
}

func shouldBeDeletedLocally(t gobdd.StepTest, ctx gobdd.Context) {
	addExpectation(t, ctx, func(t gobdd.StepTest, ctx gobdd.Context) {
		syncer, syncedFile := unwrapContext(ctx)
		if !syncer.stateData.FileStateData[syncedFile.FriendlyPath].DeletedLocal {
			iface, _ := ctx.Get(gobdd.TestingTKey{})
			testingT := iface.(*testing.T)
			testingT.Fatal()
		}
	})
}

func remoteDataShouldBeEmpty(t gobdd.StepTest, ctx gobdd.Context) {
	addExpectation(t, ctx,  func(t gobdd.StepTest, ctx gobdd.Context) {
		iface, _ := ctx.Get("remoteStateData")
		remoteStateData := iface.(*RemoteStateData)
		if len(remoteStateData.FileStateData) > 0 {
			iface, _ = ctx.Get(gobdd.TestingTKey{})
			testingT := iface.(*testing.T)
			testingT.Fatal()
		}
	})
}

func remoteDataShouldHaveFile(t gobdd.StepTest, ctx gobdd.Context, filePath string) {
	addExpectation(t, ctx, func(t gobdd.StepTest, ctx gobdd.Context) {
		iface, _ := ctx.Get("remoteStateData")
		remoteStateData := iface.(*RemoteStateData)
		if _, ok := remoteStateData.FileStateData[filePath]; !ok {
			iface, _ = ctx.Get(gobdd.TestingTKey{})
			testingT := iface.(*testing.T)
			testingT.Fatal()
		}
	})
}

func nothing(t gobdd.StepTest, ctx gobdd.Context) {
	// easy peasy
}

func addCommonSetup(suite *gobdd.Suite) {
	// See convertTime() for possible values of {time}
	suite.AddParameterTypes(`{time}`, []string{`"([\d\w\-\:\s]+)"`})
	suite.AddParameterTypes(`{filePath}`, []string{`"([\d\w\-/~\s]+)"`})
	// local file
	suite.AddStep(`the file exists locally`, fileExistsLocally)
	suite.AddStep(`the file does not exist locally`, fileDoesntExistLocally)
	suite.AddStep(`the local modified time is {time}`, localModifiedTime)
	suite.AddStep(`the last cloud update was {time}`, lastCloudUpdate)
	suite.AddStep(`the file was marked deleted locally`, wasMarkedDeletedLocally)
	suite.AddStep(`the global config has file {filePath}`, globalConfigHasFile)
	// cloud file
	suite.AddStep(`the file exists in the cloud`, fileExistsInCloud)
	suite.AddStep(`the file does not exist in the cloud`, fileDoesntExistInCloud)
	suite.AddStep(`the cloud modified time is {time}`, cloudModifiedTime)
	suite.AddStep(`the cloud has file {filePath}`, cloudHasFile)
	suite.AddStep(`the remote state data file does not exist`, remoteStateDataFileDoesNotExist)
	// actions/results
	suite.AddStep(`the file should be updated to the cloud`, fileUpdatedCloud)
	suite.AddStep(`the file should be created in the cloud`, fileUpdatedCloud)
	suite.AddStep(`the file should be downloaded from the cloud`, fileDownloadedFromCloud)
	suite.AddStep(`the file should be marked deleted locally`, shouldBeDeletedLocally)
	suite.AddStep(`nothing should happen`, nothing)
	suite.AddStep(`the remote state data should be empty`, remoteDataShouldBeEmpty)
	suite.AddStep(`the remote state data should have file {filePath}`, remoteDataShouldHaveFile)
}

func TestHandleFile(t *testing.T) {
	syncedFile := utils.SyncedFile{
		FriendlyPath: "~/test_file1",
		RealPath:     "/home/chris/test_file1",
	}
	syncer := &Syncer{
		stateData: &LocalStateData{
			FileStateData: map[string]*LocalFileStateData{},
		},
	}
	suite := gobdd.NewSuite(t, gobdd.WithBeforeScenario(func(ctx gobdd.Context) {
		ctrl := gomock.NewController(t)
		syncer.RemoteFileStore = NewMockFileStore(ctrl)
		syncer.LocalFileStore = NewMockFileStore(ctrl)
		syncer.stateData.FileStateData[syncedFile.FriendlyPath] = &LocalFileStateData{}
		ctx.Set("syncer", syncer)
		ctx.Set("syncedFile", syncedFile)
		expectations = []assertExpectationFunc{}
	}), gobdd.WithAfterScenario(func(ctx gobdd.Context) {
		syncer.handleFile(syncedFile.FriendlyPath)
		for _, assertExpectation := range expectations {
			assertExpectation(t, ctx)
		}
	}), gobdd.WithFeaturesPath("features/handleFile.feature"))
	addCommonSetup(suite)
	suite.Run()
}

func TestCleanupRemoteFiles(t *testing.T) {
	syncer := &Syncer{
		stateData: &LocalStateData{
			FileStateData: map[string]*LocalFileStateData{},
		},
	}
	suite := gobdd.NewSuite(t, gobdd.WithBeforeScenario(func(ctx gobdd.Context) {
		ctrl := gomock.NewController(t)
		remoteFileStore := NewMockFileStore(ctrl)
		syncer.RemoteFileStore = remoteFileStore
		syncer.LocalFileStore = NewMockFileStore(ctrl)
		ctx.Set("syncer", syncer)
		globalConfig = &GlobalConfig{
			TagPaths: map[string][]string{
				"all": {},
			},
		}
		remoteFiles = []utils.StoredFile{}
		remoteFileStore.EXPECT().
			WriteFileContents(gomock.Eq(utils.SyncedFile{
				FriendlyPath: stateRemoteFileFile,
			}), gomock.Any())
		expectations = []assertExpectationFunc{}
	}), gobdd.WithAfterScenario(func(ctx gobdd.Context) {
		remoteStateData, _ := syncer.cleanupRemoteFiles(remoteFiles, globalConfig)
		ctx.Set("remoteStateData", remoteStateData)
		for _, assertExpectation := range expectations {
			assertExpectation(t, ctx)
		}
	}), gobdd.WithFeaturesPath("features/cleanupRemoteFiles.feature"))
	addCommonSetup(suite)
	suite.Run()
}
