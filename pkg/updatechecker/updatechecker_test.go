package updatechecker

import (
	"strings"
	"testing"

	"github.com/blang/semver"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/cursor"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/updatechecker/types"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/require"
)

func TestAutoDeployDoesNotExecuteIfDisabled(t *testing.T) {
	var autoDeployType = apptypes.AutoDeployDisabled
	var opts = types.CheckForUpdatesOpts{}

	err := autoDeploy(opts, "cluster-id", autoDeployType)
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted to nil", err)
	}
}

func TestAutoDeployDoesNotExecuteIfNotSet(t *testing.T) {
	var opts = types.CheckForUpdatesOpts{}
	var clusterID = "some-cluster-id"

	err := autoDeploy(opts, clusterID, "")
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted to nil", err)
	}
}

func TestAutoDeployFailedToGetAppVersionsErrors(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, errors.New("app version error"))

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil && !strings.Contains(err.Error(), "app version error") {
		t.Errorf("autoDeploy() returned error = %v, wanted to include %s", err, "app version error")
	}
	if err != nil && !strings.Contains(err.Error(), "failed to get app versions for app") {
		t.Errorf("autoDeploy() returned error = %v, wanted to include %s", err, "failed to get app versions for app")
	}
}

func TestAutoDeployAppVersionsIsEmptyErrors(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		AllVersions: []*downstreamtypes.DownstreamVersion{},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil && !strings.Contains(err.Error(), "no app versions found for app "+appID) {
		t.Errorf("autoDeploy() returned error = %v, wanted to include %s", err, "no app versions found for app "+appID)
	}
}

func TestAutoDeployCurrentVersionIsNilDoesNothing(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: nil,
		AllVersions: []*downstreamtypes.DownstreamVersion{
			&downstreamtypes.DownstreamVersion{},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted nil", err)
	}
}

func TestAutoDeployCurrentVersionSemverIsNilDoesNothing(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Semver: nil,
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			&downstreamtypes.DownstreamVersion{},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted nil", err)
	}
}

func TestAutoDeploySequenceQuitsIfCurrentVersionSequenceIsGreaterThanOrEqualToMostRecent(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySequence
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var currentSequence = int64(1)
	var upgradeSequence = int64(1)
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Semver:   &semver.Version{},
			Sequence: currentSequence,
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			&downstreamtypes.DownstreamVersion{
				Sequence: upgradeSequence,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted nil", err)
	}
}

func TestAutoDeploySequenceDeploysSequenceUpgradeIfCurrentVersionLessThanMostRecent(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySequence
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var currentCursor = cursor.MustParse("1")
	var upgradeCursor = cursor.MustParse("2")
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Cursor:   &currentCursor,
			Sequence: 1,
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			{
				Cursor:   &upgradeCursor,
				Sequence: 2,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)
	mockStore.EXPECT().GetApp(appID).Return(nil, errors.New("quitting early so as not to test the waitForPreflightsToFinish method"))

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil && !strings.Contains(err.Error(), "quitting early so as not to test the waitForPreflightsToFinish method") {
		t.Errorf("autoDeploy() returned error = %v, wanted %s", err, "quitting early so as not to test the waitForPreflightsToFinish method")
	}
}

func TestAutoDeploySequenceDoesNotDeployIfCurrentVersionIsSameUpstream(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySequence
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var currentCursor = cursor.MustParse("2")
	var upgradeCursor = cursor.MustParse("2")
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Cursor:   &currentCursor,
			Sequence: 1,
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			{
				Cursor:   &upgradeCursor,
				Sequence: 2,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted nil", err)
	}
}

func TestAutoDeploySemverRequiredAllVersionsIndexIsNil(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Semver: &semver.Version{},
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{nil},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)
	// do not call waitForPreflightsToFinish

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted nil", err)
	}
}

func TestAutoDeploySemverRequiredAllVersionsHasNilSemver(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Semver: &semver.Version{},
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			{
				Semver: nil,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)
	// do not call waitForPreflightsToFinish

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted nil", err)
	}
}

func TestAutoDeploySemverRequiredNoNewVersionToDeploy(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var major = uint64(1)
	var minor = uint64(2)
	var patch = uint64(1)
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Semver: &semver.Version{
				Major: major,
				Minor: minor,
				Patch: patch,
			},
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			{
				Semver: &semver.Version{
					Major: major,
					Minor: minor,
					Patch: patch,
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)
	// do not call waitForPreflightsToFinish

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted nil", err)
	}
}

func TestAutoDeploySemverRequiredPatchUpdateMajorsDontMatch(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var currentMajor = uint64(1)
	var updateMajor = uint64(2)
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Semver: &semver.Version{
				Major: currentMajor,
			},
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			{
				Semver: &semver.Version{
					Major: updateMajor,
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)
	// do not call waitForPreflightsToFinish

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted nil", err)
	}
}

func TestAutoDeploySemverRequiredPatchUpdateMajorsMatchMinorsDontMatch(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var major = uint64(1)
	var currentMinor = uint64(2)
	var updateMinor = uint64(2)
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Semver: &semver.Version{
				Major: major,
				Minor: currentMinor,
			},
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			{
				Semver: &semver.Version{
					Major: major,
					Minor: updateMinor,
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)
	// do not call waitForPreflightsToFinish

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted nil", err)
	}
}

func TestAutoDeploySemverRequiredPatchUpdateMajorsMatchMinorsMatchWillUpgrade(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var sequence = int64(0)
	var major = uint64(1)
	var minor = uint64(2)
	var currentPatch = uint64(1)
	var upgradePatch = uint64(2)
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Semver: &semver.Version{
				Major: major,
				Minor: minor,
				Patch: currentPatch,
			},
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			{
				Semver: &semver.Version{
					Major: major,
					Minor: minor,
					Patch: upgradePatch,
				},
				Sequence: sequence,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)
	mockStore.EXPECT().GetApp(appID).Return(nil, errors.New("quitting early so as not to test the waitForPreflightsToFinish method"))

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil && !strings.Contains(err.Error(), "quitting early so as not to test the waitForPreflightsToFinish method") {
		t.Errorf("autoDeploy() returned error = %v, wanted %s", err, "quitting early so as not to test the waitForPreflightsToFinish method")
	}
}

func TestAutoDeploySemverRequiredMinorUpdateMajorsDontMatch(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverMinorPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var sequence = int64(0)
	var currentMajor = uint64(1)
	var upgradeMajor = uint64(2)
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Semver: &semver.Version{
				Major: currentMajor,
			},
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			{
				Semver: &semver.Version{
					Major: upgradeMajor,
				},
				Sequence: sequence,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)
	// do not call waitForPreflightsToFinish

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil {
		t.Errorf("autoDeploy() returned error = %v, wanted nil", err)
	}
}

func TestAutoDeploySemverRequiredMinorUpdateMajorsMatchWillUpgrade(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverMinorPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var sequence = int64(0)
	var major = uint64(1)
	var currentMinor = uint64(1)
	var upgradeMinor = uint64(2)
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Semver: &semver.Version{
				Major: major,
				Minor: currentMinor,
			},
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			{
				Semver: &semver.Version{
					Major: major,
					Minor: upgradeMinor,
				},
				Sequence: sequence,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)
	mockStore.EXPECT().GetApp(appID).Return(nil, errors.New("quitting early so as not to test the waitForPreflightsToFinish method"))

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil && !strings.Contains(err.Error(), "quitting early so as not to test the waitForPreflightsToFinish method") {
		t.Errorf("autoDeploy() returned error = %v, wanted %s", err, "quitting early so as not to test the waitForPreflightsToFinish method")
	}
}

func TestAutoDeploySemverRequiredMajorUpdateWillUpgrade(t *testing.T) {
	var autoDeployType = apptypes.AutoDeploySemverMajorMinorPatch
	var appID = "some-app"
	var clusterID = "some-cluster-id"
	var sequence = int64(0)
	var currentMajor = uint64(1)
	var upgradeMajor = uint64(2)
	var opts = types.CheckForUpdatesOpts{AppID: appID}
	var downstreamVersions = &downstreamtypes.DownstreamVersions{
		CurrentVersion: &downstreamtypes.DownstreamVersion{
			Semver: &semver.Version{
				Major: currentMajor,
			},
		},
		AllVersions: []*downstreamtypes.DownstreamVersion{
			{
				Semver: &semver.Version{
					Major: upgradeMajor,
				},
				Sequence: sequence,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersions(opts.AppID, clusterID, true).Return(downstreamVersions, nil)
	mockStore.EXPECT().GetApp(appID).Return(nil, errors.New("quitting early so as not to test the waitForPreflightsToFinish method"))

	store = mockStore

	err := autoDeploy(opts, clusterID, autoDeployType)
	if err != nil && !strings.Contains(err.Error(), "quitting early so as not to test the waitForPreflightsToFinish method") {
		t.Errorf("autoDeploy() returned error = %v, wanted %s", err, "quitting early so as not to test the waitForPreflightsToFinish method")
	}
}

func TestWaitForPreflightsToFinishGetAppErrors(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetApp(appID).Return(nil, errors.New("get app error"))

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err != nil && !strings.Contains(err.Error(), "get app error") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "get app error")
	}

	if err != nil && !strings.Contains(err.Error(), "failed get app to check for preflights") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "failed get app to check for preflights")
	}
}

func TestWaitForPreflightsToFinishAppWithoutPreflightsDoesntWait(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetApp(appID).Return(&apptypes.App{
		HasPreflight: false,
	}, nil)

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err != nil {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted nil", err)
	}
}

func TestWaitForPreflightsToFinishGetDownstreamStatusErrors(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetApp(appID).Return(&apptypes.App{
		HasPreflight: true,
	}, nil)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(storetypes.VersionPendingPreflight, errors.New("downstream error"))

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err != nil && !strings.Contains(err.Error(), "downstream error") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "downstream error")
	}

	if err != nil && !strings.Contains(err.Error(), "failed to poll for preflights results") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "failed to poll for preflights results")
	}
}

func TestWaitForPreflightsToFinishGetPreflightsResultsErrors(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetApp(appID).Return(&apptypes.App{
		HasPreflight: true,
	}, nil)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(storetypes.VersionDeployed, nil) // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(&preflighttypes.PreflightResult{
		Result: "",
	}, errors.New("preflight results error"))

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err != nil && !strings.Contains(err.Error(), "preflight results error") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "preflight results error")
	}

	if err != nil && !strings.Contains(err.Error(), "failed to fetch preflight results") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "failed to fetch preflight results")
	}
}

func TestWaitForPreflightsToFinishNoPreflightResults(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetApp(appID).Return(&apptypes.App{
		HasPreflight: true,
	}, nil)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(storetypes.VersionDeployed, nil) // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(nil, nil)

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err != nil && !strings.Contains(err.Error(), "failed to find a preflight spec") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "failed to find a preflight spec")
	}
}

func TestWaitForPreflightsToFinishPreflightResultsEmpty(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetApp(appID).Return(&apptypes.App{
		HasPreflight: true,
	}, nil)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(storetypes.VersionDeployed, nil) // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(&preflighttypes.PreflightResult{
		Result: "",
	}, nil)

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err != nil && !strings.Contains(err.Error(), "failed to find a preflight spec") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "failed to find a preflight spec")
	}
}

func TestWaitForPreflightsToFinishIsNotAnUploadPreflightResultsObject(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetApp(appID).Return(&apptypes.App{
		HasPreflight: true,
	}, nil)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(storetypes.VersionDeployed, nil) // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(&preflighttypes.PreflightResult{
		Result: "invalid",
	}, nil)

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err != nil && !strings.Contains(err.Error(), "failed to parse preflight results") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "failed to parse preflight results")
	}
}

func TestWaitForPreflightsToFinishPreflightStateContainsFailures(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetApp(appID).Return(&apptypes.App{
		HasPreflight: true,
	}, nil)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(storetypes.VersionDeployed, nil) // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(&preflighttypes.PreflightResult{
		Result: `{
					"results": [{"title":"some-title", "message": "some-message", "uri": "some-uri"}],
                    "errors": [{"error":"some-error"}]
				}`,
	}, nil)

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err != nil && !strings.Contains(err.Error(), "errors in the preflight state results") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "errors in the preflight state results")
	}
}

func TestWaitForPreflightsToFinishNoErrors(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetApp(appID).Return(&apptypes.App{
		HasPreflight: true,
	}, nil)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(storetypes.VersionDeployed, nil) // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(&preflighttypes.PreflightResult{
		Result: `{
					"results": [{"title":"some-title", "message": "some-message", "uri": "some-uri"}]
				}`,
		CreatedAt:                  nil,
		AppSlug:                    "",
		ClusterSlug:                "",
		Skipped:                    false,
		HasFailingStrictPreflights: false,
	}, nil)

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err != nil {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted nil", err)
	}
}

func TestWaitForPreflightsToFinishPreflightsNotInitiallyFinished(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetApp(appID).Return(&apptypes.App{
		HasPreflight: true,
	}, nil)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(storetypes.VersionPendingPreflight, nil) // polling
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(storetypes.VersionPendingPreflight, nil) // polling
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(storetypes.VersionDeployed, nil)         // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(&preflighttypes.PreflightResult{
		Result: `{
					"results": [{"title":"some-title", "message": "some-message", "uri": "some-uri"}]
				}`,
		CreatedAt:                  nil,
		AppSlug:                    "",
		ClusterSlug:                "",
		Skipped:                    false,
		HasFailingStrictPreflights: false,
	}, nil)

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err != nil {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted nil", err)
	}
}

func Test_removeOldUpdates(t *testing.T) {
	tests := []struct {
		useSemver   bool
		updates     []upstreamtypes.Update
		appVersions *downstreamtypes.DownstreamVersions
		want        []upstreamtypes.Update
	}{
		{
			useSemver: false,
			updates: []upstreamtypes.Update{
				{
					Cursor:       "8",
					VersionLabel: "0.0.41",
				},
				{
					Cursor:       "7",
					VersionLabel: "0.1.9",
				},
				{
					Cursor:       "6",
					VersionLabel: "0.0.32",
				},
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						VersionLabel: "0.1.5",
						UpdateCursor: "6",
						Sequence:     1,
					},
					{
						VersionLabel: "0.1.4",
						UpdateCursor: "5",
						Sequence:     0,
					},
				},
			},
			want: []upstreamtypes.Update{
				{
					Cursor:       "8",
					VersionLabel: "0.0.41",
				},
				{
					Cursor:       "7",
					VersionLabel: "0.1.9",
				},
				{
					Cursor:       "6",
					VersionLabel: "0.0.32",
				},
			},
		},
		{
			useSemver: true,
			updates: []upstreamtypes.Update{
				{
					Cursor:       "8",
					VersionLabel: "0.0.41",
				},
				{
					Cursor:       "7",
					VersionLabel: "0.1.9",
				},
				{
					Cursor:       "6",
					VersionLabel: "0.0.32",
				},
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						VersionLabel: "0.1.5",
						UpdateCursor: "6",
						Sequence:     1,
					},
					{
						VersionLabel: "0.1.4",
						UpdateCursor: "5",
						Sequence:     0,
					},
				},
			},
			want: []upstreamtypes.Update{
				{
					Cursor:       "7",
					VersionLabel: "0.1.9",
				},
			},
		},
	}

	for _, test := range tests {
		req := require.New(t)

		for _, v := range test.appVersions.AllVersions {
			sv := semver.MustParse(v.VersionLabel)
			v.Semver = &sv

			vc := cursor.MustParse(v.UpdateCursor)
			v.Cursor = &vc
		}

		got := removeOldUpdates(test.updates, test.appVersions, test.useSemver)
		req.Equal(test.want, got)
	}
}
