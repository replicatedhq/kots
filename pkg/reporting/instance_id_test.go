package reporting

import (
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/api/reporting/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_resolveAppInstanceID(t *testing.T) {
	t.Run("returns the stored instance id and immediate restore parent", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStore := mock_store.NewMockStore(ctrl)

		mockStore.EXPECT().GetAppInstanceID("app-1").Return("instance-c", []string{"instance-a", "instance-b"}, nil)

		instanceID, restoredFrom := resolveAppInstanceID(mockStore, "app-1")
		assert.Equal(t, "instance-c", instanceID)
		assert.Equal(t, "instance-b", restoredFrom)
	})

	t.Run("no lineage means no restore parent", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStore := mock_store.NewMockStore(ctrl)

		mockStore.EXPECT().GetAppInstanceID("app-1").Return("app-1", nil, nil)

		instanceID, restoredFrom := resolveAppInstanceID(mockStore, "app-1")
		assert.Equal(t, "app-1", instanceID)
		assert.Equal(t, "", restoredFrom)
	})

	t.Run("falls back to the app id on error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStore := mock_store.NewMockStore(ctrl)

		mockStore.EXPECT().GetAppInstanceID("app-1").Return("", nil, errors.New("db down"))

		instanceID, restoredFrom := resolveAppInstanceID(mockStore, "app-1")
		assert.Equal(t, "app-1", instanceID)
		assert.Equal(t, "", restoredFrom)
	})
}

func Test_restoredFromInstanceIDHeader(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	reportingInfo := &types.ReportingInfo{
		InstanceID:             "instance-b",
		RestoredFromInstanceID: "instance-a",
	}
	InjectReportingInfoHeaders(req.Header, reportingInfo)
	assert.Equal(t, "instance-b", req.Header.Get("X-Replicated-InstanceID"))
	assert.Equal(t, "instance-a", req.Header.Get("X-Replicated-RestoredFromInstanceID"))

	// header is omitted entirely when the instance was not restored
	req, err = http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	InjectReportingInfoHeaders(req.Header, &types.ReportingInfo{InstanceID: "instance-b"})
	_, present := req.Header["X-Replicated-Restoredfrominstanceid"]
	assert.False(t, present)
	assert.Empty(t, req.Header.Get("X-Replicated-RestoredFromInstanceID"))
}

func Test_buildInstanceReportIncludesRestoredFrom(t *testing.T) {
	reportingInfo := &types.ReportingInfo{
		InstanceID:             "instance-b",
		ClusterID:              "cluster-1",
		RestoredFromInstanceID: "instance-a",
		Downstream: types.DownstreamInfo{
			Cursor: "42",
		},
	}

	report := BuildInstanceReport("license-id", reportingInfo)
	require.Len(t, report.Events, 1)
	assert.Equal(t, "instance-b", report.Events[0].InstanceID)
	assert.Equal(t, "instance-a", report.Events[0].RestoredFromInstanceID)
}
