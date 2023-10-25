package reporting

import (
	"context"
	"testing"

	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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
				UserAgent:                 "KOTS/1.100.0",
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

	encodedInstanceReport, err := EncodeReport(testInstanceReport)
	req.NoError(err)

	decodedInstanceReport, err := DecodeReport(encodedInstanceReport, testInstanceReport.GetType())
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
				UserAgent:       "KOTS/1.100.0",
			},
		},
	}

	encodedPreflightReport, err := EncodeReport(testPrelightReport)
	req.NoError(err)

	decodedPreflightReport, err := DecodeReport(encodedPreflightReport, testPrelightReport.GetType())
	req.NoError(err)

	req.Equal(testPrelightReport, decodedPreflightReport)
}

func Test_CreateReportEvent(t *testing.T) {
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
				UserAgent:                 "KOTS/1.100.0",
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

	// preflight report
	testPreflightReport := &PreflightReport{
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
				UserAgent:       "KOTS/1.100.0",
			},
		},
	}

	tests := append(createTestsForEvent(t, testInstanceReport), createTestsForEvent(t, testPreflightReport)...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			err := AppendReport(tt.args.clientset, tt.args.namespace, tt.args.appSlug, tt.args.report)
			if tt.wantErr {
				req.Error(err)
				return
			}
			req.NoError(err)

			// validate secret exists and has the expected data
			secret, err := tt.args.clientset.CoreV1().Secrets(tt.args.namespace).Get(context.TODO(), tt.args.report.GetSecretName(tt.args.appSlug), metav1.GetOptions{})
			req.NoError(err)
			req.NotNil(secret.Data[tt.args.report.GetSecretKey()])

			report, err := DecodeReport(secret.Data[tt.args.report.GetSecretKey()], tt.args.report.GetType())
			req.NoError(err)
			req.Equal(tt.args.report, report)
		})
	}
}

type CreateReportEventTest struct {
	name          string
	args          CreateReportEventTestArgs
	wantNumEvents int
	wantErr       bool
}

type CreateReportEventTestArgs struct {
	clientset kubernetes.Interface
	namespace string
	appSlug   string
	report    Report
}

func createTestsForEvent(t *testing.T, testReport Report) []CreateReportEventTest {
	testReportWithOneEventData, err := EncodeReport(testReport)
	require.NoError(t, err)

	for i := 0; i < testReport.GetEventLimit(); i++ {
		err := testReport.AppendEvents(testReport)
		require.NoError(t, err)
	}
	testReportWithMaxEventsData, err := EncodeReport(testReport)
	require.NoError(t, err)

	tests := []CreateReportEventTest{
		{
			name: "secret does not exist",
			args: CreateReportEventTestArgs{
				clientset: fake.NewSimpleClientset(),
				namespace: "default",
				appSlug:   "test-app-slug",
				report:    testReport,
			},
			wantNumEvents: 1,
		},
		{
			name: "secret exists with an existing event",
			args: CreateReportEventTestArgs{
				clientset: fake.NewSimpleClientset(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testReport.GetSecretName("test-app-slug"),
							Namespace: "default",
							Labels:    kotsadmtypes.GetKotsadmLabels(),
						},
						Data: map[string][]byte{
							testReport.GetSecretKey(): testReportWithOneEventData,
						},
					},
				),
				namespace: "default",
				appSlug:   "test-app-slug",
				report:    testReport,
			},
			wantNumEvents: 2,
		},
		{
			name: "secret exists without data",
			args: CreateReportEventTestArgs{
				clientset: fake.NewSimpleClientset(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testReport.GetSecretName("test-app-slug"),
							Namespace: "default",
							Labels:    kotsadmtypes.GetKotsadmLabels(),
						},
					},
				),
				namespace: "default",
				appSlug:   "test-app-slug",
				report:    testReport,
			},
			wantNumEvents: 1,
		},
		{
			name: "secret exists with max number of events",
			args: CreateReportEventTestArgs{
				clientset: fake.NewSimpleClientset(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testReport.GetSecretName("test-app-slug"),
							Namespace: "default",
							Labels:    kotsadmtypes.GetKotsadmLabels(),
						},
						Data: map[string][]byte{
							testReport.GetSecretKey(): testReportWithMaxEventsData,
						},
					},
				),
				namespace: "default",
				appSlug:   "test-app-slug",
				report:    testReport,
			},
			wantNumEvents: ReportEventLimit,
		},
	}

	return tests
}
