package reporting

import (
	"net/http"
	"testing"

	"github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectReportingInfoHeaders(t *testing.T) {
	reportingInfo := &types.ReportingInfo{
		InstanceID:         "test-instance",
		ClusterID:          "test-cluster",
		AppStatus:          "ready",
		IsKurl:             true,
		KurlNodeCountTotal: 3,
		KurlNodeCountReady: 2,
		K8sVersion:         "1.23.0",
		K8sDistribution:    "eks",
		UserAgent:          "KOTS/1.0",
		KOTSInstallID:      "kots-123",
		KURLInstallID:      "kurl-456",
		IsGitOpsEnabled:    true,
		GitOpsProvider:     "github",
		Downstream: types.DownstreamInfo{
			Cursor:             "123",
			ChannelID:          "channel-abc",
			ChannelName:        "stable",
			Status:             "deployed",
			PreflightState:     "success",
			SkipPreflights:     false,
			ReplHelmInstalls:   2,
			NativeHelmInstalls: 1,
		},
	}

	// Create a new HTTP header
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	// Call the function being tested
	InjectReportingInfoHeaders(req.Header, reportingInfo)

	// Assert that the headers are set in the request
	assert.Equal(t, "KOTS/1.0", req.Header.Get("User-Agent"))
}
