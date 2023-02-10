package snapshot

import (
	"context"
	"reflect"
	"testing"

	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	coretest "k8s.io/client-go/testing"
)

func TestPrepareIncludedNamespaces(t *testing.T) {
	tests := []struct {
		name       string
		namespaces []string
		want       []string
	}{
		{
			name:       "empty",
			namespaces: []string{},
			want:       []string{},
		},
		{
			name:       "single",
			namespaces: []string{"test"},
			want:       []string{"test"},
		},
		{
			name:       "multiple",
			namespaces: []string{"test", "test2"},
			want:       []string{"test", "test2"},
		},
		{
			name:       "multiple ignore order",
			namespaces: []string{"test", "test2"},
			want:       []string{"test2", "test"},
		},
		{
			name:       "duplicates",
			namespaces: []string{"test", "test2", "test"},
			want:       []string{"test", "test2"},
		},
		{
			name:       "multiple with empty string",
			namespaces: []string{"test", "", "test2"},
			want:       []string{"test", "test2"},
		},
		{
			name:       "single wildcard",
			namespaces: []string{"*"},
			want:       []string{"*"},
		},
		{
			name:       "wildcard with specific",
			namespaces: []string{"*", "test"},
			want:       []string{"*"},
		},
		{
			name:       "wildcard with empty string",
			namespaces: []string{"*", ""},
			want:       []string{"*"},
		},
		{
			name:       "wildcard with empty string and specific",
			namespaces: []string{"*", "", "test"},
			want:       []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prepareIncludedNamespaces(tt.namespaces)
			if !assert.ElementsMatch(t, tt.want, got) {
				t.Errorf("prepareIncludedNamespaces() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mockGetNamespacesErrorClient() kubernetes.Interface {
	mockClient := &fake.Clientset{}
	mockClient.Fake.AddReactor("list", "namespaces", func(action coretest.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, kuberneteserrors.NewGone("kotsadm-gitops")
	})
	return mockClient
}

func mockGetNamespacesAndPodsClient() kubernetes.Interface {
	mockClient := &fake.Clientset{}
	mockClient.Fake.AddReactor("list", "namespaces", func(action coretest.Action) (handled bool, ret runtime.Object, err error) {
		return true, &corev1.NamespaceList{
			Items: []corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
			},
		}, nil
	})
	mockClient.Fake.AddReactor("list", "pods", func(action coretest.Action) (handled bool, ret runtime.Object, err error) {
		return true, &corev1.PodList{
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
						Labels: map[string]string{
							kotsadmtypes.KotsadmKey:  kotsadmtypes.KotsadmLabelValue,
							kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				},
			},
		}, nil
	})
	return mockClient
}

func mockK8sClientWithShutdownPods() kubernetes.Interface {
	mockClient := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shutdown",
				Namespace: "test",
				Labels: map[string]string{
					kotsadmtypes.KotsadmKey:  kotsadmtypes.KotsadmLabelValue,
					kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
				},
			},
			Status: corev1.PodStatus{
				Phase:  "Failed",
				Reason: "Shutdown",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-running",
				Namespace: "test",
				Labels: map[string]string{
					kotsadmtypes.KotsadmKey:  kotsadmtypes.KotsadmLabelValue,
					kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
				},
			},
			Status: corev1.PodStatus{
				Phase: "Running",
			},
		},
	)

	return mockClient
}

func Test_getNamespaces(t *testing.T) {
	type args struct {
		ctx       context.Context
		clientset kubernetes.Interface
	}
	tests := []struct {
		name           string
		args           args
		wantNamespaces []string
		wantErr        bool
	}{
		{
			name: "expect error when k8s client returns error",
			args: args{
				ctx:       context.TODO(),
				clientset: mockGetNamespacesErrorClient(),
			},
			wantNamespaces: nil,
			wantErr:        true,
		},
		{
			name: "expect no error when k8s client returns namespaces",
			args: args{
				ctx:       context.TODO(),
				clientset: mockGetNamespacesAndPodsClient(),
			},
			wantNamespaces: []string{"test"},
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNamespaces, err := getNamespaces(tt.args.ctx, tt.args.clientset)
			if (err != nil) != tt.wantErr {
				t.Errorf("getNamespaces() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotNamespaces, tt.wantNamespaces) {
				t.Errorf("getNamespaces() = %v, want %v", gotNamespaces, tt.wantNamespaces)
			}
		})
	}
}

func Test_excludeShutdownPodsFromBackup(t *testing.T) {
	type args struct {
		ctx       context.Context
		clientset kubernetes.Interface
		backup    *velerov1.Backup
	}
	tests := []struct {
		name                               string
		args                               args
		wantErr                            bool
		wantNumOfPodsWithExcludeAnnotation int
	}{
		{
			name: "expect no error when namespaces is empty",
			args: args{
				ctx:       context.TODO(),
				clientset: mockGetNamespacesAndPodsClient(),
				backup:    &velerov1.Backup{},
			},
			wantErr: false,
		},
		{
			name: "expect error when getting namespaces returns error",
			args: args{
				ctx:       context.TODO(),
				clientset: mockGetNamespacesErrorClient(),
				backup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{"*"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "expect no error when getting listing namespaces and pods with running status returns no errors",
			args: args{
				ctx:       context.TODO(),
				clientset: mockGetNamespacesAndPodsClient(),
				backup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{"*"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "expect no error when k8s client list namespaces, pods and updates pods with velero exclude annotation",
			args: args{
				ctx:       context.TODO(),
				clientset: mockK8sClientWithShutdownPods(),
				backup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{"*"},
					},
				},
			},
			wantErr:                            false,
			wantNumOfPodsWithExcludeAnnotation: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := excludeShutdownPodsFromBackup(tt.args.ctx, tt.args.clientset, tt.args.backup); (err != nil) != tt.wantErr {
				t.Errorf("excludeShutdownPodsFromBackup() error = %v, wantErr %v", err, tt.wantErr)
			}
			foundNumofPodsWithExcludeAnnotation := 0
			if !tt.wantErr {
				// get pods in test namespace and check if they have the velero exclude annotation for Shutdown pods
				pods, err := tt.args.clientset.CoreV1().Pods("test").List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					t.Errorf("excludeShutdownPodsFromBackup() error = %v, wantErr %v", err, tt.wantErr)
				}
				for _, pod := range pods.Items {
					if _, ok := pod.Labels["velero.io/exclude-from-backup"]; ok {
						foundNumofPodsWithExcludeAnnotation++
					}
					if pod.Status.Phase == corev1.PodFailed && pod.Status.Reason == "Shutdown" {
						if _, ok := pod.Labels["velero.io/exclude-from-backup"]; !ok {
							t.Errorf("excludeShutdownPodsFromBackup() velero.io/exclude-from-backup annotation not found on pod %s", pod.Name)
						}
					} else {
						if _, ok := pod.Labels["velero.io/exclude-from-backup"]; ok {
							t.Errorf("excludeShutdownPodsFromBackup() velero.io/exclude-from-backup annotation found on pod %s", pod.Name)
						}
					}
				}

				if foundNumofPodsWithExcludeAnnotation != tt.wantNumOfPodsWithExcludeAnnotation {
					t.Errorf("excludeShutdownPodsFromBackup() found %d pods with velero.io/exclude-from-backup annotation, want %d", foundNumofPodsWithExcludeAnnotation, tt.wantNumOfPodsWithExcludeAnnotation)
				}
			}
		})
	}
}
