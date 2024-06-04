package snapshot

import (
	"context"
	"testing"

	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		isEC       bool
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
		{
			name:       "wildcard with embedded cluster",
			namespaces: []string{"*", "test"},
			want:       []string{"*"},
			isEC:       true,
		},
		{
			name:       "embedded-cluster install",
			namespaces: []string{"test", "abcapp"},
			want:       []string{"test", "abcapp", "embedded-cluster", "kube-system", "openebs", "registry", "seaweedfs"},
			isEC:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prepareIncludedNamespaces(tt.namespaces, tt.isEC)
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

var appSlugMatchExpression = &metav1.LabelSelector{
	MatchExpressions: []metav1.LabelSelectorRequirement{
		{
			Key:      "kots.io/app-slug",
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{"abc-slug", "test-slug", "xyz-slug"},
		},
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
			name: "expect no error when shutdown pods are found and updated for kotsadm backup label and namespace is *",
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
		{
			name: "expect no error when shutdown pods are found and updated for app slug match expression",
			args: args{
				ctx:       context.TODO(),
				clientset: mockK8sClientWithShutdownPods(),
				veleroBackup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{"test"},
						LabelSelector:      appSlugMatchExpression,
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

func Test_excludeShutdownPodsFromBackup_check(t *testing.T) {
	res := []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "different-app-test-pod",
				Namespace: "test",
				Labels: map[string]string{
					"kots.io/app-slug": "not-test-slug",
				},
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				Phase:  "Failed",
				Reason: "Shutdown",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-included-app-test-pod",
				Namespace: "test",
				Labels: map[string]string{
					"kots.io/app-slug": "abc-slug",
				},
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				Phase:  "Failed",
				Reason: "Shutdown",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "running-test-pod",
				Namespace: "test",
				Labels: map[string]string{
					"kots.io/app-slug": "test-slug",
				},
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				Phase: "Running",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "already-labelled-test-pod",
				Namespace: "test",
				Labels: map[string]string{
					"kots.io/app-slug":              "test-slug",
					"velero.io/exclude-from-backup": "true",
				},
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				Phase:  "Failed",
				Reason: "Shutdown",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "needs-label-test-pod",
				Namespace: "test",
				Labels: map[string]string{
					"kots.io/app-slug": "test-slug",
				},
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				Phase:  "Failed",
				Reason: "Shutdown",
			},
		},
	}

	type args struct {
		veleroBackup *velerov1.Backup
	}
	tests := []struct {
		name         string
		args         args
		resources    []runtime.Object
		wantExcluded []string
	}{
		{
			name:         "expect label selector to work",
			wantExcluded: []string{"already-labelled-test-pod", "needs-label-test-pod"},
			args: args{
				veleroBackup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{"test"},
						LabelSelector:      appSlugLabelSelector,
					},
				},
			},
			resources: res,
		},
		{
			name:         "expect match expression to work",
			wantExcluded: []string{"other-included-app-test-pod", "already-labelled-test-pod", "needs-label-test-pod"},
			args: args{
				veleroBackup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{"test"},
						LabelSelector:      appSlugMatchExpression,
					},
				},
			},
			resources: res,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			mockClient := fake.NewSimpleClientset(tt.resources...)

			err := excludeShutdownPodsFromBackup(context.TODO(), mockClient, tt.args.veleroBackup)
			req.NoError(err)

			// count the number of pods with exclude annotation
			testPods, err := mockClient.CoreV1().Pods("test").List(context.TODO(), metav1.ListOptions{})
			req.NoError(err)

			foundExcluded := []string{}
			for _, pod := range testPods.Items {
				if _, ok := pod.Labels["velero.io/exclude-from-backup"]; ok {
					foundExcluded = append(foundExcluded, pod.Name)
				}
			}

			req.ElementsMatch(tt.wantExcluded, foundExcluded)
		})
	}
}

func Test_instanceBackupLabelSelectors(t *testing.T) {
	tests := []struct {
		name              string
		isEmbeddedCluster bool
		want              []*metav1.LabelSelector
	}{
		{
			name:              "not embedded cluster",
			isEmbeddedCluster: false,
			want: []*metav1.LabelSelector{
				{
					MatchLabels: map[string]string{
						"kots.io/backup": "velero",
					},
				},
			},
		},
		{
			name:              "embedded cluster",
			isEmbeddedCluster: true,
			want: []*metav1.LabelSelector{
				{
					MatchLabels: map[string]string{},
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "replicated.com/disaster-recovery",
							Operator: metav1.LabelSelectorOpIn,
							Values: []string{
								"infra",
								"app",
								"ec-install",
							},
						},
					},
				},
				{
					MatchLabels: map[string]string{
						"app": "docker-registry",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got := instanceBackupLabelSelectors(tt.isEmbeddedCluster)
			req.ElementsMatch(tt.want, got)
		})
	}
}
