package snapshot

import (
	"context"
	"testing"

	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
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

func mockGetPodsInANamespaceErrorClient() kubernetes.Interface {
	mockClient := &fake.Clientset{}
	mockClient.Fake.AddReactor("list", "pods", func(action coretest.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, kuberneteserrors.NewGone("kotsadm-backup-shutdown")
	})
	return mockClient
}

func mockUpdateShutdownPodErrorClient() kubernetes.Interface {
	mockClient := &fake.Clientset{}
	mockClient.Fake.AddReactor("list", "pods", func(action coretest.Action) (handled bool, ret runtime.Object, err error) {
		return true, &corev1.PodList{
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-backup-shutdown",
						Namespace: "test",
						Labels: map[string]string{
							"kots.io/app-slug":       "test-slug",
							kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
						},
					},
					Status: corev1.PodStatus{
						Phase:  "Failed",
						Reason: "Shutdown",
					},
				},
			},
		}, nil
	})
	//  add reactor update pod failed
	mockClient.Fake.AddReactor("update", "pods", func(action coretest.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, kuberneteserrors.NewGone("kotsadm-backup-shutdown")
	})
	return mockClient
}

func mockGetRunningPodsClient() kubernetes.Interface {
	mockClient := &fake.Clientset{}
	mockClient.Fake.AddReactor("list", "pods", func(action coretest.Action) (handled bool, ret runtime.Object, err error) {
		return true, &corev1.PodList{
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
						Labels: map[string]string{
							"kots.io/app-slug":       "test-slug",
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
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-2",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shutdown",
				Namespace: "test",
				Labels: map[string]string{
					"kots.io/app-slug":       "test-slug",
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
				Name:      "test-backup-shutdown",
				Namespace: "test-2",
				Labels: map[string]string{
					"kots.io/app-slug":       "test-slug",
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
					"kots.io/app-slug":       "test-slug",
					kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
				},
			},
			Status: corev1.PodStatus{
				Phase: "Running",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-backup-running",
				Namespace: "test-2",
				Labels: map[string]string{
					"kots.io/app-slug":       "test-slug",
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

var selectorMap = map[string]string{
	"status.phase": string(corev1.PodFailed),
}

var kotsadmBackupLabelSelector = &metav1.LabelSelector{
	MatchLabels: map[string]string{
		kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
	},
}

var kotsadmPodListOption = metav1.ListOptions{
	LabelSelector: labels.SelectorFromSet(kotsadmBackupLabelSelector.MatchLabels).String(),
	FieldSelector: fields.SelectorFromSet(selectorMap).String(),
}

var appSlugLabelSelector = &metav1.LabelSelector{
	MatchLabels: map[string]string{
		"kots.io/app-slug": "test-slug",
	},
}

var appSlugPodListOption = metav1.ListOptions{
	LabelSelector: labels.SelectorFromSet(appSlugLabelSelector.MatchLabels).String(),
	FieldSelector: fields.SelectorFromSet(selectorMap).String(),
}

func Test_excludeShutdownPodsFromBackupInNamespace(t *testing.T) {

	type args struct {
		ctx                  context.Context
		clientset            kubernetes.Interface
		namespace            string
		failedPodListOptions metav1.ListOptions
	}
	tests := []struct {
		name                               string
		args                               args
		wantErr                            bool
		wantNumOfPodsWithExcludeAnnotation int
	}{

		{
			name: "expect error when k8s client list pod returns error",
			args: args{
				ctx:                  context.TODO(),
				clientset:            mockGetPodsInANamespaceErrorClient(),
				namespace:            "test",
				failedPodListOptions: kotsadmPodListOption,
			},
			wantErr: true,
		},
		{
			name: "expect error when k8s client update shutdown pod returns error",
			args: args{
				ctx:                  context.TODO(),
				clientset:            mockUpdateShutdownPodErrorClient(),
				namespace:            "test",
				failedPodListOptions: kotsadmPodListOption,
			},
			wantErr: true,
		},
		{
			name: "expect no error when no shutdown pods are found",
			args: args{
				ctx:                  context.TODO(),
				clientset:            mockGetRunningPodsClient(),
				namespace:            "test",
				failedPodListOptions: kotsadmPodListOption,
			},
			wantErr:                            false,
			wantNumOfPodsWithExcludeAnnotation: 0,
		},
		{
			name: "expect no error when shutdown pods are found and updated for kotsadm backup label",
			args: args{
				ctx:                  context.TODO(),
				clientset:            mockK8sClientWithShutdownPods(),
				namespace:            "test",
				failedPodListOptions: kotsadmPodListOption,
			},
			wantErr:                            false,
			wantNumOfPodsWithExcludeAnnotation: 1,
		},
		{
			name: "expect no error when shutdown pods are found and updated for app slug label",
			args: args{
				ctx:                  context.TODO(),
				clientset:            mockK8sClientWithShutdownPods(),
				namespace:            "test-2",
				failedPodListOptions: appSlugPodListOption,
			},
			wantErr:                            false,
			wantNumOfPodsWithExcludeAnnotation: 1,
		},
		{
			name: "expect no error when shutdown pods are found and updated for app slug label with all namespaces",
			args: args{
				ctx:                  context.TODO(),
				clientset:            mockK8sClientWithShutdownPods(),
				namespace:            "",
				failedPodListOptions: appSlugPodListOption,
			},
			wantErr:                            false,
			wantNumOfPodsWithExcludeAnnotation: 2,
		},
		{
			name: "expect no error when shutdown pods are found and updated for kotsadm backup label with all namespaces",
			args: args{
				ctx:                  context.TODO(),
				clientset:            mockK8sClientWithShutdownPods(),
				namespace:            "",
				failedPodListOptions: kotsadmPodListOption,
			},
			wantErr:                            false,
			wantNumOfPodsWithExcludeAnnotation: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := excludeShutdownPodsFromBackupInNamespace(tt.args.ctx, tt.args.clientset, tt.args.namespace, tt.args.failedPodListOptions); (err != nil) != tt.wantErr {
				t.Errorf("excludeShutdownPodsFromBackupInNamespace() error = %v, wantErr %v", err, tt.wantErr)
			}

			foundNumofPodsWithExcludeAnnotation := 0
			if !tt.wantErr {
				// get pods in test namespace and check if they have the velero exclude annotation for Shutdown pods
				pods, err := tt.args.clientset.CoreV1().Pods(tt.args.namespace).List(context.TODO(), tt.args.failedPodListOptions)
				if err != nil {
					t.Errorf("excludeShutdownPodsFromBackupInNamespace() error = %v, wantErr %v", err, tt.wantErr)
				}
				for _, pod := range pods.Items {
					if _, ok := pod.Labels["velero.io/exclude-from-backup"]; ok {
						foundNumofPodsWithExcludeAnnotation++
					}
					if pod.Status.Phase == corev1.PodFailed && pod.Status.Reason == "Shutdown" {
						if _, ok := pod.Labels["velero.io/exclude-from-backup"]; !ok {
							t.Errorf("excludeShutdownPodsFromBackupInNamespace() velero.io/exclude-from-backup annotation not found on pod %s", pod.Name)
						}
					} else {
						if _, ok := pod.Labels["velero.io/exclude-from-backup"]; ok {
							t.Errorf("excludeShutdownPodsFromBackupInNamespace() velero.io/exclude-from-backup annotation found on pod %s", pod.Name)
						}
					}
				}

				if foundNumofPodsWithExcludeAnnotation != tt.wantNumOfPodsWithExcludeAnnotation {
					t.Errorf("excludeShutdownPodsFromBackupInNamespace() found %d pods with velero.io/exclude-from-backup annotation, want %d", foundNumofPodsWithExcludeAnnotation, tt.wantNumOfPodsWithExcludeAnnotation)
				}
			}
		})
	}
}

func Test_excludeShutdownPodsFromBackup(t *testing.T) {

	type args struct {
		ctx          context.Context
		clientset    kubernetes.Interface
		veleroBackup *velerov1.Backup
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "expect no error when namespaces are empty",
			args: args{
				ctx:       context.TODO(),
				clientset: mockK8sClientWithShutdownPods(),
				veleroBackup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{},
						LabelSelector:      kotsadmBackupLabelSelector,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "expect no error when pods are running",
			args: args{
				ctx:       context.TODO(),
				clientset: mockGetRunningPodsClient(),
				veleroBackup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{"test"},
						LabelSelector:      appSlugLabelSelector,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "expect error when k8s client list pods returns error",
			args: args{
				ctx:       context.TODO(),
				clientset: mockGetPodsInANamespaceErrorClient(),
				veleroBackup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{"test"},
						LabelSelector:      appSlugLabelSelector,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "expect no error when shutdown pods are found and updated for app slug label",
			args: args{
				ctx:       context.TODO(),
				clientset: mockK8sClientWithShutdownPods(),
				veleroBackup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{"test"},
						LabelSelector:      appSlugLabelSelector,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "expect no error when shutdown pods are found and updated for kotsadm backup label and nanespace is *",
			args: args{
				ctx:       context.TODO(),
				clientset: mockK8sClientWithShutdownPods(),
				veleroBackup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{"*"},
						LabelSelector:      appSlugLabelSelector,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := excludeShutdownPodsFromBackup(tt.args.ctx, tt.args.clientset, tt.args.veleroBackup); (err != nil) != tt.wantErr {
				t.Errorf("excludeShutdownPodsFromBackup() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
