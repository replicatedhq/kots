package reporting

import (
	"context"
	"math/rand"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_EncodeDecodeReport(t *testing.T) {
	req := require.New(t)

	var input Report

	// instance report
	input = &InstanceReport{
		Events: []InstanceReportEvent{
			createTestInstanceEvent(1234567890),
		},
	}

	encoded, err := EncodeReport(input)
	req.NoError(err)

	decoded, err := DecodeReport(encoded, input.GetType())
	req.NoError(err)

	req.Equal(input, decoded)

	// preflight report
	input = &PreflightReport{
		Events: []PreflightReportEvent{
			createTestPreflightEvent(1234567890),
		},
	}

	encoded, err = EncodeReport(input)
	req.NoError(err)

	decoded, err = DecodeReport(encoded, input.GetType())
	req.NoError(err)

	req.Equal(input, decoded)
}

func Test_AppendReport(t *testing.T) {
	req := require.New(t)

	instanceReportWithMaxEvents := getTestInstanceReportWithMaxEvents()
	instanceReportWithMaxSize, err := getTestInstanceReportWithMaxSize()
	req.NoError(err)

	preflightReportWithMaxEvents := getTestPreflightReportWithMaxEvents()
	preflightReportWithMaxSize, err := getTestPreflightReportWithMaxSize()
	req.NoError(err)

	tests := []struct {
		name           string
		appSlug        string
		existingReport Report
		newReport      Report
		wantReport     Report
	}{
		{
			name:           "instance report - no existing report",
			appSlug:        "test-app-slug",
			existingReport: nil,
			newReport: &InstanceReport{
				Events: []InstanceReportEvent{
					createTestInstanceEvent(1),
					createTestInstanceEvent(2),
					createTestInstanceEvent(3),
				},
			},
			wantReport: &InstanceReport{
				Events: []InstanceReportEvent{
					createTestInstanceEvent(1),
					createTestInstanceEvent(2),
					createTestInstanceEvent(3),
				},
			},
		},
		{
			name:    "instance report - report exists with no events",
			appSlug: "test-app-slug",
			existingReport: &InstanceReport{
				Events: []InstanceReportEvent{},
			},
			newReport: &InstanceReport{
				Events: []InstanceReportEvent{
					createTestInstanceEvent(1),
					createTestInstanceEvent(2),
					createTestInstanceEvent(3),
				},
			},
			wantReport: &InstanceReport{
				Events: []InstanceReportEvent{
					createTestInstanceEvent(1),
					createTestInstanceEvent(2),
					createTestInstanceEvent(3),
				},
			},
		},
		{
			name:    "instance report - report exists with a few events",
			appSlug: "test-app-slug",
			existingReport: &InstanceReport{
				Events: []InstanceReportEvent{
					createTestInstanceEvent(1),
					createTestInstanceEvent(2),
					createTestInstanceEvent(3),
				},
			},
			newReport: &InstanceReport{
				Events: []InstanceReportEvent{
					createTestInstanceEvent(4),
					createTestInstanceEvent(5),
					createTestInstanceEvent(6),
				},
			},
			wantReport: &InstanceReport{
				Events: []InstanceReportEvent{
					createTestInstanceEvent(1),
					createTestInstanceEvent(2),
					createTestInstanceEvent(3),
					createTestInstanceEvent(4),
					createTestInstanceEvent(5),
					createTestInstanceEvent(6),
				},
			},
		},
		{
			name:           "instance report - report exists with max number of events",
			appSlug:        "test-app-slug",
			existingReport: instanceReportWithMaxEvents,
			newReport: &InstanceReport{
				Events: []InstanceReportEvent{
					createTestInstanceEvent(int64(instanceReportWithMaxEvents.GetEventLimit())),
					createTestInstanceEvent(int64(instanceReportWithMaxEvents.GetEventLimit() + 1)),
					createTestInstanceEvent(int64(instanceReportWithMaxEvents.GetEventLimit() + 2)),
				},
			},
			wantReport: &InstanceReport{
				Events: append(instanceReportWithMaxEvents.Events[3:], []InstanceReportEvent{
					createTestInstanceEvent(int64(instanceReportWithMaxEvents.GetEventLimit())),
					createTestInstanceEvent(int64(instanceReportWithMaxEvents.GetEventLimit() + 1)),
					createTestInstanceEvent(int64(instanceReportWithMaxEvents.GetEventLimit() + 2)),
				}...),
			},
		},
		{
			name:           "instance report - report exists with max report size",
			appSlug:        "test-app-slug",
			existingReport: instanceReportWithMaxSize,
			newReport: &InstanceReport{
				Events: []InstanceReportEvent{
					createLargeTestInstanceEvent(int64(len(instanceReportWithMaxSize.Events))),
					createLargeTestInstanceEvent(int64(len(instanceReportWithMaxSize.Events) + 1)),
					createLargeTestInstanceEvent(int64(len(instanceReportWithMaxSize.Events) + 2)),
				},
			},
			wantReport: &InstanceReport{
				Events: append(instanceReportWithMaxSize.Events[3:], []InstanceReportEvent{
					createLargeTestInstanceEvent(int64(len(instanceReportWithMaxSize.Events))),
					createLargeTestInstanceEvent(int64(len(instanceReportWithMaxSize.Events) + 1)),
					createLargeTestInstanceEvent(int64(len(instanceReportWithMaxSize.Events) + 2)),
				}...),
			},
		},
		{
			name:           "preflight report - no existing report",
			appSlug:        "test-app-slug",
			existingReport: nil,
			newReport: &PreflightReport{
				Events: []PreflightReportEvent{
					createTestPreflightEvent(1),
					createTestPreflightEvent(2),
					createTestPreflightEvent(3),
				},
			},
			wantReport: &PreflightReport{
				Events: []PreflightReportEvent{
					createTestPreflightEvent(1),
					createTestPreflightEvent(2),
					createTestPreflightEvent(3),
				},
			},
		},
		{
			name:    "preflight report - report exists with no events",
			appSlug: "test-app-slug",
			existingReport: &PreflightReport{
				Events: []PreflightReportEvent{},
			},
			newReport: &PreflightReport{
				Events: []PreflightReportEvent{
					createTestPreflightEvent(1),
					createTestPreflightEvent(2),
					createTestPreflightEvent(3),
				},
			},
			wantReport: &PreflightReport{
				Events: []PreflightReportEvent{
					createTestPreflightEvent(1),
					createTestPreflightEvent(2),
					createTestPreflightEvent(3),
				},
			},
		},
		{
			name:    "preflight report - report exists with a few events",
			appSlug: "test-app-slug",
			existingReport: &PreflightReport{
				Events: []PreflightReportEvent{
					createTestPreflightEvent(1),
					createTestPreflightEvent(2),
					createTestPreflightEvent(3),
				},
			},
			newReport: &PreflightReport{
				Events: []PreflightReportEvent{
					createTestPreflightEvent(4),
					createTestPreflightEvent(5),
					createTestPreflightEvent(6),
				},
			},
			wantReport: &PreflightReport{
				Events: []PreflightReportEvent{
					createTestPreflightEvent(1),
					createTestPreflightEvent(2),
					createTestPreflightEvent(3),
					createTestPreflightEvent(4),
					createTestPreflightEvent(5),
					createTestPreflightEvent(6),
				},
			},
		},
		{
			name:           "preflight report - report exists with max number of events",
			appSlug:        "test-app-slug",
			existingReport: preflightReportWithMaxEvents,
			newReport: &PreflightReport{
				Events: []PreflightReportEvent{
					createTestPreflightEvent(int64(preflightReportWithMaxEvents.GetEventLimit())),
					createTestPreflightEvent(int64(preflightReportWithMaxEvents.GetEventLimit() + 1)),
					createTestPreflightEvent(int64(preflightReportWithMaxEvents.GetEventLimit() + 2)),
				},
			},
			wantReport: &PreflightReport{
				Events: append(preflightReportWithMaxEvents.Events[3:], []PreflightReportEvent{
					createTestPreflightEvent(int64(preflightReportWithMaxEvents.GetEventLimit())),
					createTestPreflightEvent(int64(preflightReportWithMaxEvents.GetEventLimit() + 1)),
					createTestPreflightEvent(int64(preflightReportWithMaxEvents.GetEventLimit() + 2)),
				}...),
			},
		},
		{
			name:           "preflight report - report exists with max report size",
			appSlug:        "test-app-slug",
			existingReport: preflightReportWithMaxSize,
			newReport: &PreflightReport{
				Events: []PreflightReportEvent{
					createLargeTestPreflightEvent(int64(len(preflightReportWithMaxSize.Events))),
					createLargeTestPreflightEvent(int64(len(preflightReportWithMaxSize.Events) + 1)),
					createLargeTestPreflightEvent(int64(len(preflightReportWithMaxSize.Events) + 2)),
				},
			},
			wantReport: &PreflightReport{
				Events: append(preflightReportWithMaxSize.Events[3:], []PreflightReportEvent{
					createLargeTestPreflightEvent(int64(len(preflightReportWithMaxSize.Events))),
					createLargeTestPreflightEvent(int64(len(preflightReportWithMaxSize.Events) + 1)),
					createLargeTestPreflightEvent(int64(len(preflightReportWithMaxSize.Events) + 2)),
				}...),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			if tt.existingReport != nil {
				encoded, err := EncodeReport(tt.existingReport)
				req.NoError(err)

				clientset = fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tt.existingReport.GetSecretName(tt.appSlug),
						Namespace: "default",
					},
					Data: map[string][]byte{
						tt.existingReport.GetSecretKey(): encoded,
					},
				})
			}

			err := AppendReport(clientset, "default", tt.appSlug, tt.newReport)
			req.NoError(err)

			// validate secret exists and has the expected data
			secret, err := clientset.CoreV1().Secrets("default").Get(context.TODO(), tt.wantReport.GetSecretName(tt.appSlug), metav1.GetOptions{})
			req.NoError(err)
			req.NotNil(secret.Data[tt.wantReport.GetSecretKey()])

			gotReport, err := DecodeReport(secret.Data[tt.wantReport.GetSecretKey()], tt.wantReport.GetType())
			req.NoError(err)

			if tt.wantReport.GetType() == ReportTypeInstance {
				wantNumOfEvents := len(tt.wantReport.(*InstanceReport).Events)
				gotNumOfEvents := len(gotReport.(*InstanceReport).Events)

				if wantNumOfEvents != gotNumOfEvents {
					t.Errorf("want %d events, got %d", wantNumOfEvents, gotNumOfEvents)
					return
				}

				req.Equal(tt.wantReport, gotReport)
			} else {
				wantNumOfEvents := len(tt.wantReport.(*PreflightReport).Events)
				gotNumOfEvents := len(gotReport.(*PreflightReport).Events)

				if wantNumOfEvents != gotNumOfEvents {
					t.Errorf("want %d events, got %d", wantNumOfEvents, gotNumOfEvents)
					return
				}

				req.Equal(tt.wantReport, gotReport)
			}
		})
	}
}

func createTestInstanceEvent(reportedAt int64) InstanceReportEvent {
	testDownstreamSequence := int64(123)
	return InstanceReportEvent{
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
		EmbeddedClusterID:         "test-embedded-cluster-id",
		EmbeddedClusterVersion:    "test-embedded-cluster-version",
		IsGitOpsEnabled:           true,
		GitOpsProvider:            "test-gitops-provider",
		SnapshotProvider:          "test-snapshot-provider",
		SnapshotFullSchedule:      "0 0 * * *",
		SnapshotFullTTL:           "720h",
		SnapshotPartialSchedule:   "0 0 * * *",
		SnapshotPartialTTL:        "720h",
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
}

func createLargeTestInstanceEvent(seed int64) InstanceReportEvent {
	r := rand.New(rand.NewSource(seed))

	sizeInBytes := 100 * 1024 // 100KB

	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

	randomBytes := make([]byte, sizeInBytes)
	for i := 0; i < sizeInBytes; i++ {
		randomBytes[i] = charset[r.Intn(len(charset))]
	}

	return InstanceReportEvent{
		InstallStatus: string(randomBytes), // can use any field here
	}
}

func getTestInstanceReportWithMaxEvents() *InstanceReport {
	report := &InstanceReport{
		Events: []InstanceReportEvent{},
	}
	for i := 0; i < report.GetEventLimit(); i++ {
		report.Events = append(report.Events, createTestInstanceEvent(int64(i)))
	}
	return report
}

func getTestInstanceReportWithMaxSize() (*InstanceReport, error) {
	report := &InstanceReport{
		Events: []InstanceReportEvent{},
	}

	encoded, err := EncodeReport(report)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode instance report")
	}

	for i := 0; len(encoded) <= report.GetSizeLimit(); i++ {
		seed := int64(i)
		event := createLargeTestInstanceEvent(seed)
		eventSize := len(event.InstallStatus)

		if len(encoded)+eventSize > report.GetSizeLimit() {
			break
		}

		report.Events = append(report.Events, event)

		encoded, err = EncodeReport(report)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encode instance report")
		}
	}

	return report, nil
}

func createTestPreflightEvent(reportedAt int64) PreflightReportEvent {
	return PreflightReportEvent{
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
	}
}

func createLargeTestPreflightEvent(seed int64) PreflightReportEvent {
	r := rand.New(rand.NewSource(seed))

	sizeInBytes := 100 * 1024 // 100KB

	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

	randomBytes := make([]byte, sizeInBytes)
	for i := 0; i < sizeInBytes; i++ {
		randomBytes[i] = charset[r.Intn(len(charset))]
	}

	return PreflightReportEvent{
		PreflightStatus: string(randomBytes), // can use any field here
	}
}

func getTestPreflightReportWithMaxEvents() *PreflightReport {
	report := &PreflightReport{
		Events: []PreflightReportEvent{},
	}
	for i := 0; i < report.GetEventLimit(); i++ {
		report.Events = append(report.Events, createTestPreflightEvent(int64(i)))
	}
	return report
}

func getTestPreflightReportWithMaxSize() (*PreflightReport, error) {
	report := &PreflightReport{
		Events: []PreflightReportEvent{},
	}

	encoded, err := EncodeReport(report)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode preflight report")
	}

	for i := 0; len(encoded) <= report.GetSizeLimit(); i++ {
		seed := int64(i)
		event := createLargeTestPreflightEvent(seed)
		eventSize := len(event.PreflightStatus)

		if len(encoded)+eventSize > report.GetSizeLimit() {
			break
		}

		report.Events = append(report.Events, event)

		encoded, err = EncodeReport(report)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encode preflight report")
		}
	}

	return report, nil
}
