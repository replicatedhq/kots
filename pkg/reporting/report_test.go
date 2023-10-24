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
	testInstanceReport := &Report{
		Events: []ReportEvent{
			&InstanceReportEvent{
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

	encodedInstanceReport, err := EncodeReport(testInstanceReport)
	req.NoError(err)

	decodedInstanceReport := &Report{}
	err = DecodeReport(encodedInstanceReport, decodedInstanceReport, "instance")
	req.NoError(err)

	req.Equal(testInstanceReport, decodedInstanceReport)

	// preflight report
	testPrelightReport := &Report{
		Events: []ReportEvent{
			&PreflightReportEvent{
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

	encodedPreflightReport, err := EncodeReport(testPrelightReport)
	req.NoError(err)

	decodedPreflightReport := &Report{}
	err = DecodeReport(encodedPreflightReport, decodedPreflightReport, "preflight")
	req.NoError(err)

	req.Equal(testPrelightReport, decodedPreflightReport)
}

func Test_CreateReportEvent(t *testing.T) {
	// instance report
	testDownstreamSequence := int64(123)
	testInstanceReportEvent := &InstanceReportEvent{
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
	}

	testPreflightReportEvent := &PreflightReportEvent{
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
	}

	tests := append(createTestsForEvent(t, testInstanceReportEvent), createTestsForEvent(t, testPreflightReportEvent)...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			err := CreateReportEvent(tt.args.clientset, tt.args.namespace, tt.args.appSlug, tt.args.event)
			if tt.wantErr {
				req.Error(err)
				return
			}
			req.NoError(err)

			// validate secret exists and has the expected data
			secret, err := tt.args.clientset.CoreV1().Secrets(tt.args.namespace).Get(context.TODO(), tt.args.event.GetReportSecretName(tt.args.appSlug), metav1.GetOptions{})
			req.NoError(err)
			req.NotNil(secret.Data[tt.args.event.GetReportSecretKey()])

			report := &Report{}
			err = DecodeReport(secret.Data[tt.args.event.GetReportSecretKey()], report, tt.args.event.GetReportType())
			req.NoError(err)

			req.Len(report.Events, tt.wantNumEvents)

			for _, event := range report.Events {
				req.Equal(tt.args.event, event)
			}
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
	event     ReportEvent
}

func createTestsForEvent(t *testing.T, testEvent ReportEvent) []CreateReportEventTest {
	testReportWithOneEvent := &Report{
		Events: []ReportEvent{testEvent},
	}
	testReportWithOneEventData, err := EncodeReport(testReportWithOneEvent)
	require.NoError(t, err)

	// testReportWithMaxEvents := &Report{}
	// for i := 0; i < testEvent.GetReportEventLimit(); i++ {
	// 	testReportWithMaxEvents.Events = append(testReportWithMaxEvents.Events, testEvent)
	// }
	// testReportWithMaxEventsData, err := EncodeReport(testReportWithMaxEvents)
	// require.NoError(t, err)

	type args struct {
		clientset kubernetes.Interface
		namespace string
		appSlug   string
		event     ReportEvent
	}
	tests := []CreateReportEventTest{
		{
			name: "secret does not exist",
			args: CreateReportEventTestArgs{
				clientset: fake.NewSimpleClientset(),
				namespace: "default",
				appSlug:   "test-app-slug",
				event:     testEvent,
			},
			wantNumEvents: 1,
		},
		{
			name: "secret exists with an existing event",
			args: CreateReportEventTestArgs{
				clientset: fake.NewSimpleClientset(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testEvent.GetReportSecretName("test-app-slug"),
							Namespace: "default",
							Labels:    kotsadmtypes.GetKotsadmLabels(),
						},
						Data: map[string][]byte{
							testEvent.GetReportSecretKey(): testReportWithOneEventData,
						},
					},
				),
				namespace: "default",
				appSlug:   "test-app-slug",
				event:     testEvent,
			},
			wantNumEvents: 2,
		},
		{
			name: "secret exists without data",
			args: CreateReportEventTestArgs{
				clientset: fake.NewSimpleClientset(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testEvent.GetReportSecretName("test-app-slug"),
							Namespace: "default",
							Labels:    kotsadmtypes.GetKotsadmLabels(),
						},
					},
				),
				namespace: "default",
				appSlug:   "test-app-slug",
				event:     testEvent,
			},
			wantNumEvents: 1,
		},
		// {
		// 	name: "secret exists with max number of events",
		// 	args: CreateReportEventTestArgs{
		// 		clientset: fake.NewSimpleClientset(
		// 			&corev1.Secret{
		// 				ObjectMeta: metav1.ObjectMeta{
		// 					Name:      testEvent.GetReportSecretName("test-app-slug"),
		// 					Namespace: "default",
		// 					Labels:    kotsadmtypes.GetKotsadmLabels(),
		// 				},
		// 				Data: map[string][]byte{
		// 					testEvent.GetReportSecretKey(): testReportWithMaxEventsData,
		// 				},
		// 			},
		// 		),
		// 		namespace: "default",
		// 		appSlug:   "test-app-slug",
		// 		event:     testEvent,
		// 	},
		// 	wantNumEvents: testEvent.GetReportEventLimit(),
		// },
	}

	return tests
}
