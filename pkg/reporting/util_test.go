package reporting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_EncodeDecodeAirgapReport(t *testing.T) {
	req := require.New(t)

	// instance report
	testDownstreamSequence := int64(123)
	testInstanceReport := &InstanceReport{
		Events: []InstanceReportEvent{
			{
				ReportedAt:                1234567890,
				LicenseID:                 "test-license-id",
				InstanceID:                "test-instance-id",
				ClusterID:                 "test-cluster-id",
				AppStatus:                 "ready",
				IsKurl:                    true,
				KurlNodeCountTotal:        3,
				KurlNodeCountReady:        3,
				K8sVersion:                "1.28.0",
				K8sDistribution:           "kurl",
				KotsVersion:               "1.100.0",
				KotsInstallID:             "test-kots-install-id",
				KurlInstallID:             "test-kurl-install-id",
				IsGitOpsEnabled:           true,
				GitOpsProvider:            "test-gitops-provider",
				DownstreamChannelID:       "test-downstream-channel-id",
				DownstreamChannelSequence: 123,
				DownstreamChannelName:     "test-downstream-channel-name",
				DownstreamSequence:        &testDownstreamSequence,
				DownstreamSource:          "test-downstream-source",
				InstallStatus:             "installed",
				PreflightState:            "passed",
				SkipPreflights:            false,
				ReplHelmInstalls:          1,
				NativeHelmInstalls:        2,
			},
		},
	}

	encodedInstanceReport, err := EncodeAirgapReport(testInstanceReport)
	req.NoError(err)

	decodedInstanceReport := &InstanceReport{}
	err = DecodeAirgapReport(encodedInstanceReport, decodedInstanceReport)
	req.NoError(err)

	req.Equal(testInstanceReport, decodedInstanceReport)

	// preflight report
	testPrelightReport := &PreflightReport{
		Events: []PreflightReportEvent{
			{
				ReportedAt:      1234567890,
				LicenseID:       "test-license-id",
				InstanceID:      "test-instance-id",
				ClusterID:       "test-cluster-id",
				Sequence:        123,
				SkipPreflights:  false,
				InstallStatus:   "installed",
				IsCLI:           true,
				PreflightStatus: "pass",
				AppStatus:       "ready",
				KotsVersion:     "1.100.0",
			},
		},
	}

	encodedPreflightReport, err := EncodeAirgapReport(testPrelightReport)
	req.NoError(err)

	decodedPreflightReport := &PreflightReport{}
	err = DecodeAirgapReport(encodedPreflightReport, decodedPreflightReport)
	req.NoError(err)

	req.Equal(testPrelightReport, decodedPreflightReport)
}
