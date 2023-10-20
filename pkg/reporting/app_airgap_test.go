package reporting

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateInstanceReportEvent(t *testing.T) {
	testDownstreamSequence := int64(123)
	testEvent := InstanceReportEvent{
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

	testReportWithOneEvent := &InstanceReport{
		Events: []InstanceReportEvent{testEvent},
	}
	testReportWithOneEventData, err := EncodeAirgapReport(testReportWithOneEvent)
	require.NoError(t, err)

	testReportWithMaxEvents := &InstanceReport{}
	for i := 0; i < InstanceReportEventLimit; i++ {
		testReportWithMaxEvents.Events = append(testReportWithMaxEvents.Events, testEvent)
	}
	testReportWithMaxEventsData, err := EncodeAirgapReport(testReportWithMaxEvents)
	require.NoError(t, err)

	type args struct {
		clientset kubernetes.Interface
		namespace string
		appSlug   string
		event     InstanceReportEvent
	}
	tests := []struct {
		name          string
		args          args
		wantNumEvents int
		wantErr       bool
	}{
		{
			name: "secret does not exist",
			args: args{
				clientset: fake.NewSimpleClientset(&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm",
						Namespace: "default",
						UID:       "test-uid",
					},
				}),
				namespace: "default",
				appSlug:   "test-app-slug",
				event:     testEvent,
			},
			wantNumEvents: 1,
		},
		{
			name: "secret exists with an existing event",
			args: args{
				clientset: fake.NewSimpleClientset(
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kotsadm",
							Namespace: "default",
							UID:       "test-uid",
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf(InstanceReportSecretNameFormat, "test-app-slug"),
							Namespace: "default",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: "apps/v1",
									Kind:       "Deployment",
									Name:       "kotsadm",
									UID:        "test-uid",
								},
							},
						},
						Data: map[string][]byte{
							InstanceReportSecretKey: testReportWithOneEventData,
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
			args: args{
				clientset: fake.NewSimpleClientset(
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kotsadm",
							Namespace: "default",
							UID:       "test-uid",
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf(InstanceReportSecretNameFormat, "test-app-slug"),
							Namespace: "default",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: "apps/v1",
									Kind:       "Deployment",
									Name:       "kotsadm",
									UID:        "test-uid",
								},
							},
						},
					},
				),
				namespace: "default",
				appSlug:   "test-app-slug",
				event:     testEvent,
			},
			wantNumEvents: 1,
		},
		{
			name: "secret exists with max number of events",
			args: args{
				clientset: fake.NewSimpleClientset(
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kotsadm",
							Namespace: "default",
							UID:       "test-uid",
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf(InstanceReportSecretNameFormat, "test-app-slug"),
							Namespace: "default",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: "apps/v1",
									Kind:       "Deployment",
									Name:       "kotsadm",
									UID:        "test-uid",
								},
							},
						},
						Data: map[string][]byte{
							InstanceReportSecretKey: testReportWithMaxEventsData,
						},
					},
				),
				namespace: "default",
				appSlug:   "test-app-slug",
				event:     testEvent,
			},
			wantNumEvents: InstanceReportEventLimit,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			err := CreateInstanceReportEvent(tt.args.clientset, tt.args.namespace, tt.args.appSlug, tt.args.event)
			if tt.wantErr {
				req.Error(err)
				return
			}
			req.NoError(err)

			// validate secret exists and has the expected data
			secret, err := tt.args.clientset.CoreV1().Secrets(tt.args.namespace).Get(context.TODO(), fmt.Sprintf(InstanceReportSecretNameFormat, tt.args.appSlug), metav1.GetOptions{})
			req.NoError(err)
			req.NotNil(secret.Data[InstanceReportSecretKey])

			report := &InstanceReport{}
			err = DecodeAirgapReport(secret.Data[InstanceReportSecretKey], report)
			req.NoError(err)

			req.Len(report.Events, tt.wantNumEvents)

			for _, event := range report.Events {
				req.Equal(testEvent, event)
			}
		})
	}
}
