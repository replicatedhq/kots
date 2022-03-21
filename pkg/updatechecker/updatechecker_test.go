package updatechecker

import (
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/replicatedhq/kots/pkg/store/types"
	"strings"
	"testing"
)

func TestWaitForPreflightsToFinishGetDownstreamStatusErrors(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(types.VersionPendingPreflight, errors.New("downstream error"))

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err == nil || !strings.Contains(err.Error(), "downstream error") || !strings.Contains(err.Error(), "failed to poll for preflights results") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "downstream error")
	}
}

func TestWaitForPreflightsToFinishGetPreflightsResultsErrors(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(types.VersionDeployed, nil) // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(&preflighttypes.PreflightResult{
		Result:                     "",
		CreatedAt:                  nil,
		AppSlug:                    "",
		ClusterSlug:                "",
		Skipped:                    false,
		HasFailingStrictPreflights: false,
	}, errors.New("preflight results error"))

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err == nil || !strings.Contains(err.Error(), "preflight results error") || !strings.Contains(err.Error(), "failed to fetch preflight results") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "preflight results error")
	}
}

func TestWaitForPreflightsToFinishNoPreflightResults(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(types.VersionDeployed, nil) // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(nil, nil)

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err == nil || !strings.Contains(err.Error(), "failed to find a preflight spec") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "failed to find a preflight spec")
	}
}

func TestWaitForPreflightsToFinishPreflightResultsEmpty(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(types.VersionDeployed, nil) // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(&preflighttypes.PreflightResult{
		Result: `{
					"results": [],
				}`,
		CreatedAt:                  nil,
		AppSlug:                    "",
		ClusterSlug:                "",
		Skipped:                    false,
		HasFailingStrictPreflights: false,
	}, nil)

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err == nil || !strings.Contains(err.Error(), "failed to find a preflight spec") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "failed to find a preflight spec")
	}
}

func TestWaitForPreflightsToFinishIsNotAnUploadPreflightResultsObject(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(types.VersionDeployed, nil) // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(&preflighttypes.PreflightResult{
		Result:                     "invalid",
		CreatedAt:                  nil,
		AppSlug:                    "",
		ClusterSlug:                "",
		Skipped:                    false,
		HasFailingStrictPreflights: false,
	}, nil)

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err == nil || !strings.Contains(err.Error(), "failed to parse preflight results") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "failed to parse preflight results")
	}
}

func TestWaitForPreflightsToFinishErrorsInThePreflightResults(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(types.VersionDeployed, nil) // preflight done
	mockStore.EXPECT().GetPreflightResults(appID, sequence).Return(&preflighttypes.PreflightResult{
		Result: `{
					"results": [{"title":"some-title", "message": "some-message", "uri": "some-uri"}],
                    "errors": [{"error":"some-error"}]
				}`,
		CreatedAt:                  nil,
		AppSlug:                    "",
		ClusterSlug:                "",
		Skipped:                    false,
		HasFailingStrictPreflights: false,
	}, nil)

	store = mockStore

	err := waitForPreflightsToFinish(appID, sequence)
	if err == nil || !strings.Contains(err.Error(), "preflight errors") {
		t.Errorf("waitForPreflightsToFinish() returned error = %v, wanted to include %s", err, "preflight errors")
	}
}

func TestWaitForPreflightsToFinishNoErrors(t *testing.T) {
	var appID = "some-app"
	var sequence = int64(0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(types.VersionDeployed, nil) // preflight done
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
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(types.VersionPendingPreflight, nil) // polling
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(types.VersionPendingPreflight, nil) // polling
	mockStore.EXPECT().GetDownstreamVersionStatus(appID, sequence).Return(types.VersionDeployed, nil)         // preflight done
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
