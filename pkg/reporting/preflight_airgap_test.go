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

func TestCreatePreflightReportEvent(t *testing.T) {
	testEvent := PreflightReportEvent{
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

	testReportWithOneEvent := &PreflightReport{
		Events: []PreflightReportEvent{testEvent},
	}
	testReportWithOneEventData, err := EncodeAirgapReport(testReportWithOneEvent)
	require.NoError(t, err)

	testReportWithMaxEvents := &PreflightReport{}
	for i := 0; i < PreflightReportEventLimit; i++ {
		testReportWithMaxEvents.Events = append(testReportWithMaxEvents.Events, testEvent)
	}
	testReportWithMaxEventsData, err := EncodeAirgapReport(testReportWithMaxEvents)
	require.NoError(t, err)

	type args struct {
		clientset kubernetes.Interface
		namespace string
		appSlug   string
		event     PreflightReportEvent
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
							Name:      fmt.Sprintf(PreflightReportSecretNameFormat, "test-app-slug"),
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
							PreflightReportSecretKey: testReportWithOneEventData,
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
							Name:      fmt.Sprintf(PreflightReportSecretNameFormat, "test-app-slug"),
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
							Name:      fmt.Sprintf(PreflightReportSecretNameFormat, "test-app-slug"),
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
							PreflightReportSecretKey: testReportWithMaxEventsData,
						},
					},
				),
				namespace: "default",
				appSlug:   "test-app-slug",
				event:     testEvent,
			},
			wantNumEvents: PreflightReportEventLimit,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			err := CreatePreflightReportEvent(tt.args.clientset, tt.args.namespace, tt.args.appSlug, tt.args.event)
			if tt.wantErr {
				req.Error(err)
				return
			}
			req.NoError(err)

			// validate secret exists and has the expected data
			secret, err := tt.args.clientset.CoreV1().Secrets(tt.args.namespace).Get(context.TODO(), fmt.Sprintf(PreflightReportSecretNameFormat, tt.args.appSlug), metav1.GetOptions{})
			req.NoError(err)
			req.NotNil(secret.Data[PreflightReportSecretKey])

			report := &PreflightReport{}
			err = DecodeAirgapReport(secret.Data[PreflightReportSecretKey], report)
			req.NoError(err)

			req.Len(report.Events, tt.wantNumEvents)

			for _, event := range report.Events {
				req.Equal(testEvent, event)
			}
		})
	}
}
