package snapshot

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/store"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	velerofake "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/fake"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	coretest "k8s.io/client-go/testing"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
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
				Name:      "test-shutdown-no-label",
				Namespace: "test",
				Labels:    map[string]string{},
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
				Name:      "test-running-no-label",
				Namespace: "test",
				Labels:    map[string]string{},
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
				ctx:                  context.Background(),
				clientset:            mockGetPodsInANamespaceErrorClient(),
				namespace:            "test",
				failedPodListOptions: kotsadmPodListOption,
			},
			wantErr: true,
		},
		{
			name: "expect error when k8s client update shutdown pod returns error",
			args: args{
				ctx:                  context.Background(),
				clientset:            mockUpdateShutdownPodErrorClient(),
				namespace:            "test",
				failedPodListOptions: kotsadmPodListOption,
			},
			wantErr: true,
		},
		{
			name: "expect no error when no shutdown pods are found",
			args: args{
				ctx:                  context.Background(),
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
				ctx:                  context.Background(),
				clientset:            mockK8sClientWithShutdownPods(),
				namespace:            "test",
				failedPodListOptions: kotsadmPodListOption,
			},
			wantErr:                            false,
			wantNumOfPodsWithExcludeAnnotation: 1,
		},
		{
			name: "expect no error when shutdown pods are found and updated for no label selector",
			args: args{
				ctx:       context.Background(),
				clientset: mockK8sClientWithShutdownPods(),
				namespace: "test",
				failedPodListOptions: metav1.ListOptions{
					FieldSelector: fields.SelectorFromSet(selectorMap).String(),
				},
			},
			wantErr:                            false,
			wantNumOfPodsWithExcludeAnnotation: 2,
		},
		{
			name: "expect no error when shutdown pods are found and updated for app slug label",
			args: args{
				ctx:                  context.Background(),
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
				ctx:                  context.Background(),
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
				ctx:                  context.Background(),
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
				pods, err := tt.args.clientset.CoreV1().Pods(tt.args.namespace).List(context.Background(), tt.args.failedPodListOptions)
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
				ctx:       context.Background(),
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
				ctx:       context.Background(),
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
				ctx:       context.Background(),
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
				ctx:       context.Background(),
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
				ctx:       context.Background(),
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
				ctx:       context.Background(),
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
		{
			name: "expect no error when shutdown pods are found and updated for app slug label and no label selector",
			args: args{
				ctx:       context.Background(),
				clientset: mockK8sClientWithShutdownPods(),
				veleroBackup: &velerov1.Backup{
					Spec: velerov1.BackupSpec{
						IncludedNamespaces: []string{"test"},
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

			err := excludeShutdownPodsFromBackup(context.Background(), mockClient, tt.args.veleroBackup)
			req.NoError(err)

			// count the number of pods with exclude annotation
			testPods, err := mockClient.CoreV1().Pods("test").List(context.Background(), metav1.ListOptions{})
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
				{
					MatchLabels: map[string]string{
						"app.kubernetes.io/name": "seaweedfs",
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

func Test_appendECAnnotations(t *testing.T) {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	embeddedclusterv1beta1.AddToScheme(scheme)

	tests := []struct {
		name                 string
		prev                 map[string]string
		in                   embeddedclusterv1beta1.Installation
		seaweedFSS3ServiceIP string
		env                  map[string]string
		want                 map[string]string
	}{
		{
			name: "basic",
			prev: map[string]string{
				"prev-key": "prev-value",
			},
			in:                   embeddedclusterv1beta1.Installation{},
			seaweedFSS3ServiceIP: "",
			env: map[string]string{
				"EMBEDDED_CLUSTER_ID":      "embedded-cluster-id",
				"EMBEDDED_CLUSTER_VERSION": "embedded-cluster-version",
			},
			want: map[string]string{
				"prev-key":                         "prev-value",
				"kots.io/embedded-cluster":         "true",
				"kots.io/embedded-cluster-id":      "embedded-cluster-id",
				"kots.io/embedded-cluster-version": "embedded-cluster-version",
				"kots.io/embedded-cluster-is-ha":   "false",
			},
		},
		{
			name: "online ha",
			in: embeddedclusterv1beta1.Installation{
				Spec: embeddedclusterv1beta1.InstallationSpec{
					HighAvailability: true,
				},
			},
			seaweedFSS3ServiceIP: "",
			env: map[string]string{
				"EMBEDDED_CLUSTER_ID":      "embedded-cluster-id",
				"EMBEDDED_CLUSTER_VERSION": "embedded-cluster-version",
			},
			want: map[string]string{
				"kots.io/embedded-cluster":         "true",
				"kots.io/embedded-cluster-id":      "embedded-cluster-id",
				"kots.io/embedded-cluster-version": "embedded-cluster-version",
				"kots.io/embedded-cluster-is-ha":   "true",
			},
		},
		{
			name: "airgap ha",
			in: embeddedclusterv1beta1.Installation{
				Spec: embeddedclusterv1beta1.InstallationSpec{
					HighAvailability: true,
					AirGap:           true,
				},
			},
			seaweedFSS3ServiceIP: "10.96.0.10",
			env: map[string]string{
				"EMBEDDED_CLUSTER_ID":      "embedded-cluster-id",
				"EMBEDDED_CLUSTER_VERSION": "embedded-cluster-version",
			},
			want: map[string]string{
				"kots.io/embedded-cluster":                 "true",
				"kots.io/embedded-cluster-id":              "embedded-cluster-id",
				"kots.io/embedded-cluster-version":         "embedded-cluster-version",
				"kots.io/embedded-cluster-is-ha":           "true",
				"kots.io/embedded-cluster-seaweedfs-s3-ip": "10.96.0.10",
			},
		},
		{
			name: "with pod and service cidrs",
			in: embeddedclusterv1beta1.Installation{
				Spec: embeddedclusterv1beta1.InstallationSpec{
					Network: &embeddedclusterv1beta1.NetworkSpec{
						PodCIDR:     "10.128.0.0/20",
						ServiceCIDR: "10.129.0.0/20",
					},
				},
			},
			seaweedFSS3ServiceIP: "",
			env: map[string]string{
				"EMBEDDED_CLUSTER_ID":      "embedded-cluster-id",
				"EMBEDDED_CLUSTER_VERSION": "embedded-cluster-version",
			},
			want: map[string]string{
				"kots.io/embedded-cluster":              "true",
				"kots.io/embedded-cluster-id":           "embedded-cluster-id",
				"kots.io/embedded-cluster-version":      "embedded-cluster-version",
				"kots.io/embedded-cluster-is-ha":        "false",
				"kots.io/embedded-cluster-pod-cidr":     "10.128.0.0/20",
				"kots.io/embedded-cluster-service-cidr": "10.129.0.0/20",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			ecMeta := ecInstanceBackupMetadata{
				installation:         tt.in,
				seaweedFSS3ServiceIP: tt.seaweedFSS3ServiceIP,
			}
			got := appendECAnnotations(tt.prev, ecMeta)
			req.Equal(tt.want, got)
		})
	}
}

func Test_ecIncludedNamespaces(t *testing.T) {
	tests := []struct {
		name string
		in   embeddedclusterv1beta1.Installation
		want []string
	}{
		{
			name: "online",
			in:   embeddedclusterv1beta1.Installation{},
			want: []string{
				"embedded-cluster",
				"kube-system",
				"openebs",
			},
		},
		{
			name: "online ha",
			in: embeddedclusterv1beta1.Installation{
				Spec: embeddedclusterv1beta1.InstallationSpec{
					HighAvailability: true,
				},
			},
			want: []string{
				"embedded-cluster",
				"kube-system",
				"openebs",
			},
		},
		{
			name: "airgap",
			in: embeddedclusterv1beta1.Installation{
				Spec: embeddedclusterv1beta1.InstallationSpec{
					AirGap: true,
				},
			},
			want: []string{
				"embedded-cluster",
				"kube-system",
				"openebs",
				"registry",
			},
		},
		{
			name: "airgap ha",
			in: embeddedclusterv1beta1.Installation{
				Spec: embeddedclusterv1beta1.InstallationSpec{
					HighAvailability: true,
					AirGap:           true,
				},
			},
			want: []string{
				"embedded-cluster",
				"kube-system",
				"openebs",
				"registry",
				"seaweedfs",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got := ecIncludedNamespaces(tt.in)
			req.Equal(tt.want, got)
		})
	}
}

func Test_appendCommonAnnotations(t *testing.T) {
	kotsadmSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm",
			Namespace: "kotsadm",
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "kotsadm",
							Image: "kotsadm/kotsadm:1.0.0",
						},
					},
				},
			},
		},
	}
	registryCredsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registry-creds",
			Namespace: "kotsadm",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": []byte(`{"auths":{"host":{"username":"kurl","password":"password"}}}`),
		},
	}

	type args struct {
		k8sClient   kubernetes.Interface
		annotations map[string]string
		metadata    instanceBackupMetadata
		hasAppSpec  bool
	}
	tests := []struct {
		name    string
		setup   func(t *testing.T)
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "cli install, airgap, multiple apps, not scheduled, has ttl",
			setup: func(t *testing.T) {
				t.Setenv("DISABLE_OUTBOUND_CONNECTIONS", "true")
			},
			args: args{
				k8sClient:   fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				annotations: map[string]string{},
				metadata: instanceBackupMetadata{
					backupName:                     "instance-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app: &apptypes.App{},
							kotsKinds: &kotsutil.KotsKinds{
								Installation: kotsv1beta1.Installation{
									Spec: kotsv1beta1.InstallationSpec{
										VersionLabel: "1.0.1",
									},
								},
							},
							parentSequence: 1,
						},
						"app-2": {
							app: &apptypes.App{},
							kotsKinds: &kotsutil.KotsKinds{
								Installation: kotsv1beta1.Installation{
									Spec: kotsv1beta1.InstallationSpec{
										VersionLabel: "1.0.2",
									},
								},
							},
							parentSequence: 2,
						},
					},
					isScheduled: false,
					snapshotTTL: 24 * time.Hour,
					ec:          nil,
				},
				hasAppSpec: false,
			},
			want: map[string]string{
				"kots.io/apps-sequences":           "{\"app-1\":1,\"app-2\":2}",
				"kots.io/apps-versions":            "{\"app-1\":\"1.0.1\",\"app-2\":\"1.0.2\"}",
				"kots.io/embedded-registry":        "host",
				"kots.io/instance":                 "true",
				"kots.io/is-airgap":                "true",
				"kots.io/kotsadm-deploy-namespace": "kotsadm",
				"kots.io/kotsadm-image":            "kotsadm/kotsadm:1.0.0",
				"kots.io/snapshot-requested":       "2024-01-01T00:00:00Z",
				"kots.io/snapshot-trigger":         "manual",
				"replicated.com/backup-count":      "1",
			},
		},
		{
			name: "ec install, scheduled, no ttl, improved dr",
			setup: func(t *testing.T) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")
			},
			args: args{
				k8sClient:   fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				annotations: map[string]string{},
				metadata: instanceBackupMetadata{
					backupName:                     "instance-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app: &apptypes.App{},
							kotsKinds: &kotsutil.KotsKinds{
								Installation: kotsv1beta1.Installation{
									Spec: kotsv1beta1.InstallationSpec{
										VersionLabel: "1.0.1",
									},
								},
							},
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec: &ecInstanceBackupMetadata{
						installation: embeddedclusterv1beta1.Installation{
							Spec: embeddedclusterv1beta1.InstallationSpec{
								HighAvailability: true,
								Network: &embeddedclusterv1beta1.NetworkSpec{
									PodCIDR:     "10.128.0.0/20",
									ServiceCIDR: "10.129.0.0/20",
								},
								RuntimeConfig: &embeddedclusterv1beta1.RuntimeConfigSpec{
									DataDir: "/var/lib/ec",
									AdminConsole: embeddedclusterv1beta1.AdminConsoleSpec{
										Port: 30001,
									},
									LocalArtifactMirror: embeddedclusterv1beta1.LocalArtifactMirrorSpec{
										Port: 50001,
									},
								},
							},
						},
						seaweedFSS3ServiceIP: "10.96.0.10",
					},
				},
				hasAppSpec: true,
			},
			want: map[string]string{
				"kots.io/apps-sequences":                              "{\"app-1\":1}",
				"kots.io/apps-versions":                               "{\"app-1\":\"1.0.1\"}",
				"kots.io/embedded-registry":                           "host",
				"kots.io/instance":                                    "true",
				"kots.io/is-airgap":                                   "false",
				"kots.io/kotsadm-deploy-namespace":                    "kotsadm",
				"kots.io/kotsadm-image":                               "kotsadm/kotsadm:1.0.0",
				"kots.io/snapshot-requested":                          "2024-01-01T00:00:00Z",
				"kots.io/snapshot-trigger":                            "schedule",
				"replicated.com/backup-count":                         "2",
				"kots.io/embedded-cluster":                            "true",
				"kots.io/embedded-cluster-id":                         "embedded-cluster-id",
				"kots.io/embedded-cluster-version":                    "embedded-cluster-version",
				"kots.io/embedded-cluster-is-ha":                      "true",
				"kots.io/embedded-cluster-pod-cidr":                   "10.128.0.0/20",
				"kots.io/embedded-cluster-service-cidr":               "10.129.0.0/20",
				"kots.io/embedded-cluster-seaweedfs-s3-ip":            "10.96.0.10",
				"kots.io/embedded-cluster-admin-console-port":         "30001",
				"kots.io/embedded-cluster-local-artifact-mirror-port": "50001",
				"kots.io/embedded-cluster-data-dir":                   "/var/lib/ec",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			got, err := appendCommonAnnotations(tt.args.k8sClient, tt.args.annotations, tt.args.metadata, tt.args.hasAppSpec)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_mergeAppBackupSpec(t *testing.T) {
	mockStoreExpectApp1 := func(mockStore *mock_store.MockStore) {
		mockStore.EXPECT().GetLatestAppSequence("1", true).Times(1).Return(int64(1), nil)
		mockStore.EXPECT().GetRegistryDetailsForApp("1").Times(1).Return(registrytypes.RegistrySettings{
			Hostname:   "hostname",
			Username:   "username",
			Password:   "password",
			Namespace:  "namespace",
			IsReadOnly: true,
		}, nil)
	}

	type args struct {
		backup           *velerov1.Backup
		appMeta          appInstanceBackupMetadata
		kotsadmNamespace string
		isEC             bool
	}
	tests := []struct {
		name    string
		setup   func(t *testing.T, mockStore *mock_store.MockStore)
		args    args
		want    *velerov1.Backup
		wantErr bool
	}{
		{
			name: "no backup spec",
			args: args{
				backup: &velerov1.Backup{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "velero.io/v1",
						Kind:       "Backup",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:         "",
						GenerateName: "instance-",
						Annotations: map[string]string{
							"annotation": "true",
						},
					},
					Spec: velerov1.BackupSpec{
						StorageLocation:    "default",
						IncludedNamespaces: []string{"kotsadm"},
					},
				},
				appMeta: appInstanceBackupMetadata{
					app: &apptypes.App{
						ID:       "1",
						Slug:     "app-1",
						IsAirgap: true,
					},
					kotsKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AdditionalNamespaces: []string{"another-namespace-1", "another-namespace-2"},
							},
						},
						Installation: kotsv1beta1.Installation{
							Spec: kotsv1beta1.InstallationSpec{
								VersionLabel: "1.0.1",
							},
						},
					},
					parentSequence: 1,
				},
				kotsadmNamespace: "kotsadm",
				isEC:             false,
			},
			want: &velerov1.Backup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "velero.io/v1",
					Kind:       "Backup",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:         "",
					GenerateName: "instance-",
					Annotations: map[string]string{
						"annotation": "true",
					},
				},
				Spec: velerov1.BackupSpec{
					StorageLocation:    "default",
					IncludedNamespaces: []string{"kotsadm"},
				},
			},
		},
		{
			name: "has backup spec",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStoreExpectApp1(mockStore)
			},
			args: args{
				backup: &velerov1.Backup{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "velero.io/v1",
						Kind:       "Backup",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:         "",
						GenerateName: "instance-",
						Annotations: map[string]string{
							"annotation": "true",
						},
					},
					Spec: velerov1.BackupSpec{
						StorageLocation:    "default",
						IncludedNamespaces: []string{"kotsadm"},
					},
				},
				appMeta: appInstanceBackupMetadata{
					app: &apptypes.App{
						ID:       "1",
						Slug:     "app-1",
						IsAirgap: true,
					},
					kotsKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AdditionalNamespaces: []string{"another-namespace-1", "another-namespace-2"},
							},
						},
						Installation: kotsv1beta1.Installation{
							Spec: kotsv1beta1.InstallationSpec{
								VersionLabel: "1.0.1",
							},
						},
						Backup: &velerov1.Backup{
							TypeMeta: metav1.TypeMeta{
								APIVersion: "velero.io/v1",
								Kind:       "Backup",
							},
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									"annotation-1": "true",
									"annotation-2": "false",
								},
							},
							Spec: velerov1.BackupSpec{
								IncludedNamespaces: []string{"include-namespace-1", "include-namespace-2", "template-isairgap-{{repl IsAirgap }}"},
								ExcludedNamespaces: []string{"exclude-namespace-1", "exclude-namespace-2"},
								OrderedResources: map[string]string{
									"resource-1": "true",
									"resource-2": "false",
								},
								Hooks: velerov1.BackupHooks{
									Resources: []velerov1.BackupResourceHookSpec{
										{
											Name: "hook-1",
										},
										{
											Name: "hook-2",
										},
									},
								},
							},
						},
					},
					parentSequence: 1,
				},
				kotsadmNamespace: "kotsadm",
				isEC:             false,
			},
			want: &velerov1.Backup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "velero.io/v1",
					Kind:       "Backup",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:         "",
					GenerateName: "instance-",
					Annotations: map[string]string{
						"annotation":   "true",
						"annotation-1": "true",
						"annotation-2": "false",
					},
				},
				Spec: velerov1.BackupSpec{
					StorageLocation:    "default",
					IncludedNamespaces: []string{"kotsadm", "another-namespace-1", "another-namespace-2", "include-namespace-1", "include-namespace-2", "template-isairgap-true"},
					ExcludedNamespaces: []string{"exclude-namespace-1", "exclude-namespace-2"},
					OrderedResources: map[string]string{
						"resource-1": "true",
						"resource-2": "false",
					},
					Hooks: velerov1.BackupHooks{
						Resources: []velerov1.BackupResourceHookSpec{
							{
								Name: "hook-1",
							},
							{
								Name: "hook-2",
							},
						},
					},
				},
			},
		},
		{
			name: "ec, no backup spec",
			args: args{
				backup: &velerov1.Backup{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "velero.io/v1",
						Kind:       "Backup",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:         "",
						GenerateName: "instance-",
						Annotations: map[string]string{
							"annotation": "true",
						},
					},
					Spec: velerov1.BackupSpec{
						StorageLocation:    "default",
						IncludedNamespaces: []string{"kotsadm"},
					},
				},
				appMeta: appInstanceBackupMetadata{
					app: &apptypes.App{
						ID:       "1",
						Slug:     "app-1",
						IsAirgap: true,
					},
					kotsKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AdditionalNamespaces: []string{"another-namespace-1", "another-namespace-2"},
							},
						},
						Installation: kotsv1beta1.Installation{
							Spec: kotsv1beta1.InstallationSpec{
								VersionLabel: "1.0.1",
							},
						},
					},
					parentSequence: 1,
				},
				kotsadmNamespace: "kotsadm",
				isEC:             true,
			},
			want: &velerov1.Backup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "velero.io/v1",
					Kind:       "Backup",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:         "",
					GenerateName: "instance-",
					Annotations: map[string]string{
						"annotation": "true",
					},
				},
				Spec: velerov1.BackupSpec{
					StorageLocation:    "default",
					IncludedNamespaces: []string{"kotsadm", "another-namespace-1", "another-namespace-2"},
				},
			},
		},
		{
			name: "ec, has backup spec",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStoreExpectApp1(mockStore)
			},
			args: args{
				backup: &velerov1.Backup{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "velero.io/v1",
						Kind:       "Backup",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:         "",
						GenerateName: "instance-",
						Annotations: map[string]string{
							"annotation": "true",
						},
					},
					Spec: velerov1.BackupSpec{
						StorageLocation:    "default",
						IncludedNamespaces: []string{"kotsadm"},
					},
				},
				appMeta: appInstanceBackupMetadata{
					app: &apptypes.App{
						ID:       "1",
						Slug:     "app-1",
						IsAirgap: true,
					},
					kotsKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AdditionalNamespaces: []string{"another-namespace-1", "another-namespace-2"},
							},
						},
						Installation: kotsv1beta1.Installation{
							Spec: kotsv1beta1.InstallationSpec{
								VersionLabel: "1.0.1",
							},
						},
						Backup: &velerov1.Backup{
							TypeMeta: metav1.TypeMeta{
								APIVersion: "velero.io/v1",
								Kind:       "Backup",
							},
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									"annotation-1": "true",
									"annotation-2": "false",
								},
							},
							Spec: velerov1.BackupSpec{
								IncludedNamespaces: []string{"include-namespace-1", "include-namespace-2", "template-isairgap-{{repl IsAirgap }}"},
								ExcludedNamespaces: []string{"exclude-namespace-1", "exclude-namespace-2"},
								OrderedResources: map[string]string{
									"resource-1": "true",
									"resource-2": "false",
								},
								Hooks: velerov1.BackupHooks{
									Resources: []velerov1.BackupResourceHookSpec{
										{
											Name: "hook-1",
										},
										{
											Name: "hook-2",
										},
									},
								},
							},
						},
					},
					parentSequence: 1,
				},
				kotsadmNamespace: "kotsadm",
				isEC:             true,
			},
			want: &velerov1.Backup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "velero.io/v1",
					Kind:       "Backup",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:         "",
					GenerateName: "instance-",
					Annotations: map[string]string{
						"annotation":   "true",
						"annotation-1": "true",
						"annotation-2": "false",
					},
				},
				Spec: velerov1.BackupSpec{
					StorageLocation:    "default",
					IncludedNamespaces: []string{"kotsadm", "another-namespace-1", "another-namespace-2", "include-namespace-1", "include-namespace-2", "template-isairgap-true"},
					ExcludedNamespaces: []string{"exclude-namespace-1", "exclude-namespace-2"},
					OrderedResources: map[string]string{
						"resource-1": "true",
						"resource-2": "false",
					},
					Hooks: velerov1.BackupHooks{
						Resources: []velerov1.BackupResourceHookSpec{
							{
								Name: "hook-1",
							},
							{
								Name: "hook-2",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mock_store.NewMockStore(ctrl)
			store.SetStore(mockStore)

			t.Cleanup(func() {
				store.SetStore(nil)
			})

			if tt.setup != nil {
				tt.setup(t, mockStore)
			}
			err := mergeAppBackupSpec(tt.args.backup, tt.args.appMeta, tt.args.kotsadmNamespace, tt.args.isEC)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, tt.args.backup)
		})
	}
}

func Test_getAppInstanceBackupSpec(t *testing.T) {
	EnableImprovedDR = true
	t.Cleanup(func() {
		EnableImprovedDR = false
	})

	mockStoreExpectApp1 := func(mockStore *mock_store.MockStore) {
		mockStore.EXPECT().GetLatestAppSequence("1", true).Times(1).Return(int64(1), nil)
		mockStore.EXPECT().GetRegistryDetailsForApp("1").Times(1).Return(registrytypes.RegistrySettings{
			Hostname:   "hostname",
			Username:   "username",
			Password:   "password",
			Namespace:  "namespace",
			IsReadOnly: true,
		}, nil)
	}

	kotsadmSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm",
			Namespace: "kotsadm",
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "kotsadm",
							Image: "kotsadm/kotsadm:1.0.0",
						},
					},
				},
			},
		},
	}
	registryCredsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registry-creds",
			Namespace: "kotsadm",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": []byte(`{"auths":{"host":{"username":"kurl","password":"password"}}}`),
		},
	}

	app1 := &apptypes.App{
		ID:       "1",
		Slug:     "app-1",
		IsAirgap: true,
	}

	app2 := &apptypes.App{
		ID:       "2",
		Slug:     "app-2",
		IsAirgap: true,
	}

	kotsKinds := &kotsutil.KotsKinds{
		KotsApplication: kotsv1beta1.Application{
			Spec: kotsv1beta1.ApplicationSpec{
				AdditionalNamespaces: []string{"another-namespace-1", "another-namespace-2"},
			},
		},
		Installation: kotsv1beta1.Installation{
			Spec: kotsv1beta1.InstallationSpec{
				VersionLabel: "1.0.1",
			},
		},
		Backup: &velerov1.Backup{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "velero.io/v1",
				Kind:       "Backup",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-backup",
				Annotations: map[string]string{
					"annotation-1": "true",
					"annotation-2": "false",
				},
			},
			Spec: velerov1.BackupSpec{
				StorageLocation:    "blah",
				TTL:                metav1.Duration{Duration: 1 * time.Hour},
				IncludedNamespaces: []string{"include-namespace-1", "include-namespace-2", "template-isairgap-{{repl IsAirgap }}"},
				ExcludedNamespaces: []string{"exclude-namespace-1", "exclude-namespace-2"},
				OrderedResources: map[string]string{
					"resource-1": "true",
					"resource-2": "false",
				},
				Hooks: velerov1.BackupHooks{
					Resources: []velerov1.BackupResourceHookSpec{
						{
							Name: "hook-1",
						},
						{
							Name: "hook-2",
						},
					},
				},
			},
		},
		Restore: &velerov1.Restore{},
	}

	ecMeta := &ecInstanceBackupMetadata{
		installation: embeddedclusterv1beta1.Installation{
			Spec: embeddedclusterv1beta1.InstallationSpec{
				HighAvailability: true,
				Network: &embeddedclusterv1beta1.NetworkSpec{
					PodCIDR:     "10.128.0.0/20",
					ServiceCIDR: "10.129.0.0/20",
				},
				RuntimeConfig: &embeddedclusterv1beta1.RuntimeConfigSpec{
					DataDir: "/var/lib/ec",
					AdminConsole: embeddedclusterv1beta1.AdminConsoleSpec{
						Port: 30001,
					},
					LocalArtifactMirror: embeddedclusterv1beta1.LocalArtifactMirrorSpec{
						Port: 50001,
					},
				},
			},
		},
		seaweedFSS3ServiceIP: "10.96.0.10",
	}

	type args struct {
		k8sClient kubernetes.Interface
		metadata  instanceBackupMetadata
	}
	tests := []struct {
		name   string
		setup  func(t *testing.T, mockStore *mock_store.MockStore)
		args   args
		assert func(t *testing.T, got *velerov1.Backup, err error)
	}{
		{
			name: "not ec with backup and restore spec should return nil",
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app: app1,
							kotsKinds: &kotsutil.KotsKinds{
								Backup:  &velerov1.Backup{},
								Restore: &velerov1.Restore{},
							},
							parentSequence: 1,
						},
					},
					ec: nil,
				},
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Nil(t, got)
			},
		},
		{
			name: "ec wihtout restore spec should return nil",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "instance-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app: app1,
							kotsKinds: &kotsutil.KotsKinds{
								Backup: &velerov1.Backup{},
							},
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Nil(t, got)
			},
		},
		{
			name: "ec with backup and restore spec and multiple apps should return error",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "instance-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
						"app-2": {
							app:            app2,
							kotsKinds:      kotsKinds,
							parentSequence: 2,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.Error(t, err)
				assert.Nil(t, got)
			},
		},
		{
			name: "not ec with backup and restore spec and multiple apps should not return error",
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app: app1,
							kotsKinds: &kotsutil.KotsKinds{
								Backup:  &velerov1.Backup{},
								Restore: &velerov1.Restore{},
							},
							parentSequence: 1,
						},
						"app-2": {
							app: app2,
							kotsKinds: &kotsutil.KotsKinds{
								Backup:  &velerov1.Backup{},
								Restore: &velerov1.Restore{},
							},
							parentSequence: 2,
						},
					},
					ec: nil,
				},
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Nil(t, got)
			},
		},
		{
			name: "ec with backup and restore spec should override name",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Equal(t, "", got.Name)
				assert.Equal(t, "application-", got.GenerateName)
			},
		},
		{
			name: "ec with backup and restore spec should append backup name label",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				if assert.Contains(t, got.Labels, "replicated.com/backup-name") {
					assert.Equal(t, "app-1-17332487841234", got.Labels["replicated.com/backup-name"])
				}
			},
		},
		{
			name: "ec with backup and restore spec should append common annotations",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				if assert.Contains(t, got.Annotations, "replicated.com/backup-type") {
					assert.Equal(t, "app", got.Annotations["replicated.com/backup-type"])
				}
				if assert.Contains(t, got.Annotations, "replicated.com/backup-count") {
					assert.Equal(t, "2", got.Annotations["replicated.com/backup-count"])
				}
			},
		},
		{
			name: "ec with backup and restore spec overrides storage location",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "instance-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Equal(t, "default", got.Spec.StorageLocation)
			},
		},
		{
			name: "ec with backup and restore spec overrides snapshot ttl",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "instance-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					snapshotTTL: 24 * time.Hour,
					ec:          ecMeta,
				},
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Equal(t, metav1.Duration{Duration: 24 * time.Hour}, got.Spec.TTL)
			},
		},
		{
			name: "ec with backup and restore spec does not override snapshot ttl if unset",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "instance-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Equal(t, metav1.Duration{Duration: 1 * time.Hour}, got.Spec.TTL)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mock_store.NewMockStore(ctrl)
			store.SetStore(mockStore)

			t.Cleanup(func() {
				store.SetStore(nil)
			})

			if tt.setup != nil {
				tt.setup(t, mockStore)
			}
			got, err := getAppInstanceBackupSpec(tt.args.k8sClient, tt.args.metadata)
			tt.assert(t, got, err)
		})
	}
}

func Test_getInfrastructureInstanceBackupSpec(t *testing.T) {
	kotsadmSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm",
			Namespace: "kotsadm",
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "kotsadm",
							Image: "kotsadm/kotsadm:1.0.0",
						},
					},
				},
			},
		},
	}
	registryCredsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registry-creds",
			Namespace: "kotsadm",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": []byte(`{"auths":{"host":{"username":"kurl","password":"password"}}}`),
		},
	}

	app1 := &apptypes.App{
		ID:       "1",
		Slug:     "app-1",
		IsAirgap: true,
	}

	kotsKinds := &kotsutil.KotsKinds{
		KotsApplication: kotsv1beta1.Application{
			Spec: kotsv1beta1.ApplicationSpec{
				AdditionalNamespaces: []string{"another-namespace-1", "another-namespace-2", "duplicate-namespace"},
			},
		},
		Installation: kotsv1beta1.Installation{
			Spec: kotsv1beta1.InstallationSpec{
				VersionLabel: "1.0.1",
			},
		},
		Backup: &velerov1.Backup{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "velero.io/v1",
				Kind:       "Backup",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-backup",
				Annotations: map[string]string{
					"annotation-1": "true",
					"annotation-2": "false",
				},
			},
			Spec: velerov1.BackupSpec{
				StorageLocation:    "blah",
				TTL:                metav1.Duration{Duration: 1 * time.Hour},
				IncludedNamespaces: []string{"include-namespace-1", "include-namespace-2", "template-isairgap-{{repl IsAirgap }}", "duplicate-namespace"},
				ExcludedNamespaces: []string{"exclude-namespace-1", "exclude-namespace-2"},
				OrderedResources: map[string]string{
					"resource-1": "true",
					"resource-2": "false",
				},
				Hooks: velerov1.BackupHooks{
					Resources: []velerov1.BackupResourceHookSpec{
						{
							Name: "hook-1",
						},
						{
							Name: "hook-2",
						},
					},
				},
			},
		},
		Restore: &velerov1.Restore{},
	}

	ecMeta := &ecInstanceBackupMetadata{
		installation: embeddedclusterv1beta1.Installation{
			Spec: embeddedclusterv1beta1.InstallationSpec{
				HighAvailability: true,
				Network: &embeddedclusterv1beta1.NetworkSpec{
					PodCIDR:     "10.128.0.0/20",
					ServiceCIDR: "10.129.0.0/20",
				},
				RuntimeConfig: &embeddedclusterv1beta1.RuntimeConfigSpec{
					DataDir: "/var/lib/ec",
					AdminConsole: embeddedclusterv1beta1.AdminConsoleSpec{
						Port: 30001,
					},
					LocalArtifactMirror: embeddedclusterv1beta1.LocalArtifactMirrorSpec{
						Port: 50001,
					},
				},
			},
		},
		seaweedFSS3ServiceIP: "10.96.0.10",
	}

	mockStoreExpectApp1 := func(mockStore *mock_store.MockStore) {
		mockStore.EXPECT().GetLatestAppSequence("1", true).Times(1).Return(int64(1), nil)
		mockStore.EXPECT().GetRegistryDetailsForApp("1").Times(1).Return(registrytypes.RegistrySettings{
			Hostname:   "hostname",
			Username:   "username",
			Password:   "password",
			Namespace:  "namespace",
			IsReadOnly: true,
		}, nil)
	}

	type args struct {
		k8sClient  kubernetes.Interface
		metadata   instanceBackupMetadata
		hasAppSpec bool
	}
	tests := []struct {
		name   string
		setup  func(t *testing.T, mockStore *mock_store.MockStore)
		args   args
		assert func(t *testing.T, got *velerov1.Backup, err error)
	}{
		{
			name: "should append backup name label",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          nil,
				},
				hasAppSpec: false,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				if assert.Contains(t, got.Labels, "replicated.com/backup-name") {
					assert.Equal(t, "app-1-17332487841234", got.Labels["replicated.com/backup-name"])
				}
			},
		},
		{
			name: "KOTSADM_TARGET_NAMESPACE should be added to includedNamespaces",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				util.KotsadmTargetNamespace = "kotsadm-target"
				t.Cleanup(func() {
					util.KotsadmTargetNamespace = ""
				})

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          nil,
				},
				hasAppSpec: false,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Contains(t, got.Spec.IncludedNamespaces, "kotsadm")
				assert.Contains(t, got.Spec.IncludedNamespaces, "kotsadm-target")
			},
		},
		{
			name: "if kurl should be added to includedNamespaces",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kurl-config",
						Namespace: "kube-system",
					},
				}),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          nil,
				},
				hasAppSpec: false,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Contains(t, got.Spec.IncludedNamespaces, "kurl")
			},
		},
		{
			name: "not cluster scoped should include backup storage location namespace",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          nil,
				},
				hasAppSpec: false,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Contains(t, got.Spec.IncludedNamespaces, "kotsadm-backups")
			},
		},
		{
			name: "cluster scoped should not include backup storage location namespace",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret, &rbacv1.ClusterRoleBinding{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "rbac.authorization.k8s.io/v1",
						Kind:       "ClusterRoleBinding",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "kotsadm-rolebinding",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "kotsadm",
							Namespace: "kotsadm",
						},
					},
				}),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          nil,
				},
				hasAppSpec: false,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.NotContains(t, got.Spec.IncludedNamespaces, "kotsadm-backups")
			},
		},
		{
			name: "should merge backup spec when not using improved dr",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
				hasAppSpec: false,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Contains(t, got.Spec.IncludedNamespaces, "include-namespace-1")
				if assert.Contains(t, got.Annotations, "replicated.com/backup-type") {
					assert.Equal(t, "combined", got.Annotations["replicated.com/backup-type"])
				}
			},
		},
		{
			name: "should not merge backup spec when using improved dr",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
				hasAppSpec: true,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.NotContains(t, got.Spec.IncludedNamespaces, "include-namespace-1")
				if assert.Contains(t, got.Annotations, "replicated.com/backup-type") {
					assert.Equal(t, "infra", got.Annotations["replicated.com/backup-type"])
				}
			},
		},
		{
			name: "should add ec namespaces to includedNamespaces if ec",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
				hasAppSpec: false,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Contains(t, got.Spec.IncludedNamespaces, "embedded-cluster")
			},
		},
		{
			name: "should add ec namespaces to includedNamespaces if ec",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
				hasAppSpec: false,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Contains(t, got.Spec.IncludedNamespaces, "embedded-cluster")
			},
		},
		{
			name: "should override snapshot ttl if set",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					snapshotTTL: 24 * time.Hour,
					ec:          ecMeta,
				},
				hasAppSpec: false,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Equal(t, metav1.Duration{Duration: 24 * time.Hour}, got.Spec.TTL)
			},
		},
		{
			name: "should not override snapshot ttl if unset",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
				hasAppSpec: false,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				assert.Zero(t, got.Spec.TTL)
			},
		},
		{
			name: "should deduplicate includedNamespaces",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "embedded-cluster-version")

				mockStoreExpectApp1(mockStore)
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(kotsadmSts, registryCredsSecret),
				metadata: instanceBackupMetadata{
					backupName:                     "app-1-17332487841234",
					backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					kotsadmNamespace:               "kotsadm",
					backupStorageLocationNamespace: "kotsadm-backups",
					apps: map[string]appInstanceBackupMetadata{
						"app-1": {
							app:            app1,
							kotsKinds:      kotsKinds,
							parentSequence: 1,
						},
					},
					isScheduled: true,
					ec:          ecMeta,
				},
				hasAppSpec: false,
			},
			assert: func(t *testing.T, got *velerov1.Backup, err error) {
				require.NoError(t, err)
				count := 0
				for _, ns := range got.Spec.IncludedNamespaces {
					if ns == "duplicate-namespace" {
						count++
					}
				}
				assert.Equal(t, 1, count, "Duplicate namespace should be removed")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mock_store.NewMockStore(ctrl)
			store.SetStore(mockStore)

			t.Cleanup(func() {
				store.SetStore(nil)
			})

			if tt.setup != nil {
				tt.setup(t, mockStore)
			}
			got, err := getInfrastructureInstanceBackupSpec(context.Background(), tt.args.k8sClient, tt.args.metadata, tt.args.hasAppSpec)
			tt.assert(t, got, err)
		})
	}
}

func Test_getInstanceBackupMetadata(t *testing.T) {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	embeddedclusterv1beta1.AddToScheme(scheme)

	testBsl := &velerov1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "velero",
		},
		Spec: velerov1.BackupStorageLocationSpec{
			Provider: "aws",
			Default:  true,
		},
	}
	veleroNamespaceConfigmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kotsadm-velero-namespace",
		},
		Data: map[string]string{
			"veleroNamespace": "velero",
		},
	}
	veleroDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "velero",
			Namespace: "velero",
		},
	}

	installation := embeddedclusterv1beta1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name: "20060102150405",
		},
		Spec: embeddedclusterv1beta1.InstallationSpec{
			BinaryName: "my-app",
		},
	}
	seaweedFSS3Service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ec-seaweedfs-s3",
			Namespace: "seaweedfs",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.96.0.10",
		},
	}

	type args struct {
		k8sClient    kubernetes.Interface
		ctrlClient   ctrlclient.Client
		veleroClient veleroclientv1.VeleroV1Interface
		cluster      *downstreamtypes.Downstream
		isScheduled  bool
	}
	tests := []struct {
		name    string
		setup   func(t *testing.T, mockStore *mock_store.MockStore)
		args    args
		want    instanceBackupMetadata
		wantErr bool
	}{
		{
			name: "cli install",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				util.PodNamespace = "test"
				t.Cleanup(func() {
					util.PodNamespace = ""
				})

				mockStore.EXPECT().ListInstalledApps().Times(1).Return([]*apptypes.App{
					{
						ID:       "1",
						Name:     "App 1",
						Slug:     "app-1",
						IsAirgap: true,
					},
					{
						ID:       "2",
						Name:     "App 2",
						Slug:     "app-2",
						IsAirgap: true,
					},
				}, nil)
				mockStore.EXPECT().ListDownstreamsForApp(gomock.Any()).Times(2).Return([]downstreamtypes.Downstream{
					{
						ClusterID:        "cluster-id",
						ClusterSlug:      "cluster-slug",
						Name:             "cluster-name",
						CurrentSequence:  1,
						SnapshotSchedule: "manual",
						SnapshotTTL:      "24h",
					},
				}, nil)
				mockStore.EXPECT().GetCurrentParentSequence("1", "cluster-id").Times(1).Return(int64(1), nil)
				mockStore.EXPECT().GetCurrentParentSequence("2", "cluster-id").Times(1).Return(int64(2), nil)
				mockStore.EXPECT().GetAppVersionArchive("1", int64(1), gomock.Any()).Times(1).DoAndReturn(func(appID string, sequence int64, archiveDir string) error {
					err := setupArchiveDirectoriesAndFiles(archiveDir, map[string]string{
						"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-1
spec:
  title: My App 1`,
					})
					require.NoError(t, err)
					return nil
				})
				mockStore.EXPECT().GetAppVersionArchive("2", int64(2), gomock.Any()).Times(1).DoAndReturn(func(appID string, sequence int64, archiveDir string) error {
					err := setupArchiveDirectoriesAndFiles(archiveDir, map[string]string{
						"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-2
spec:
  title: My App 2`,
					})
					require.NoError(t, err)
					return nil
				})
			},
			args: args{
				k8sClient:    fake.NewSimpleClientset(veleroNamespaceConfigmap, veleroDeployment),
				ctrlClient:   ctrlclientfake.NewClientBuilder().WithScheme(scheme).WithObjects().Build(),
				veleroClient: velerofake.NewSimpleClientset(testBsl).VeleroV1(),
				cluster: &downstreamtypes.Downstream{
					SnapshotTTL: "24h",
				},
				isScheduled: true,
			},
			want: instanceBackupMetadata{
				backupName:                     "instance-",
				backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				kotsadmNamespace:               "test",
				backupStorageLocationNamespace: "velero",
				apps: map[string]appInstanceBackupMetadata{
					"app-1": {
						app: &apptypes.App{
							ID:       "1",
							Name:     "App 1",
							Slug:     "app-1",
							IsAirgap: true,
						},
						kotsKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "kots.io/v1beta1",
									Kind:       "Application",
								},
								ObjectMeta: metav1.ObjectMeta{
									Name: "app-1",
								},
								Spec: kotsv1beta1.ApplicationSpec{
									Title: "My App 1",
								},
							},
							Installation: kotsv1beta1.Installation{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "kots.io/v1beta1",
									Kind:       "Installation",
								},
							},
						},
						parentSequence: 1,
					},
					"app-2": {
						app: &apptypes.App{
							ID:       "2",
							Name:     "App 2",
							Slug:     "app-2",
							IsAirgap: true,
						},
						kotsKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "kots.io/v1beta1",
									Kind:       "Application",
								},
								ObjectMeta: metav1.ObjectMeta{
									Name: "app-2",
								},
								Spec: kotsv1beta1.ApplicationSpec{
									Title: "My App 2",
								},
							},
							Installation: kotsv1beta1.Installation{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "kots.io/v1beta1",
									Kind:       "Installation",
								},
							},
						},
						parentSequence: 2,
					},
				},
				isScheduled: true,
				snapshotTTL: 24 * time.Hour,
				ec:          nil,
			},
		},
		{
			name: "ec install",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "embedded-cluster-id")

				util.PodNamespace = "test"
				t.Cleanup(func() {
					util.PodNamespace = ""
				})

				mockStore.EXPECT().ListInstalledApps().Times(1).Return([]*apptypes.App{
					{
						ID:       "1",
						Name:     "App 1",
						Slug:     "app-1",
						IsAirgap: true,
					},
				}, nil)
				mockStore.EXPECT().ListDownstreamsForApp(gomock.Any()).Times(1).Return([]downstreamtypes.Downstream{
					{
						ClusterID:        "cluster-id",
						ClusterSlug:      "cluster-slug",
						Name:             "cluster-name",
						CurrentSequence:  1,
						SnapshotSchedule: "manual",
						SnapshotTTL:      "24h",
					},
				}, nil)
				mockStore.EXPECT().GetCurrentParentSequence("1", "cluster-id").Times(1).Return(int64(1), nil)
				mockStore.EXPECT().GetAppVersionArchive("1", int64(1), gomock.Any()).Times(1).DoAndReturn(func(appID string, sequence int64, archiveDir string) error {
					err := setupArchiveDirectoriesAndFiles(archiveDir, map[string]string{
						"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-1
spec:
  title: My App 1`,
					})
					require.NoError(t, err)
					return nil
				})
			},
			args: args{
				k8sClient: fake.NewSimpleClientset(veleroNamespaceConfigmap, veleroDeployment),
				ctrlClient: ctrlclientfake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&installation,
					seaweedFSS3Service,
				).Build(),
				veleroClient: velerofake.NewSimpleClientset(testBsl).VeleroV1(),
				cluster: &downstreamtypes.Downstream{
					SnapshotTTL: "24h",
				},
				isScheduled: true,
			},
			want: instanceBackupMetadata{
				backupName:                     "app-1-",
				backupReqestedAt:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				kotsadmNamespace:               "test",
				backupStorageLocationNamespace: "velero",
				apps: map[string]appInstanceBackupMetadata{
					"app-1": {
						app: &apptypes.App{
							ID:       "1",
							Name:     "App 1",
							Slug:     "app-1",
							IsAirgap: true,
						},
						kotsKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "kots.io/v1beta1",
									Kind:       "Application",
								},
								ObjectMeta: metav1.ObjectMeta{
									Name: "app-1",
								},
								Spec: kotsv1beta1.ApplicationSpec{
									Title: "My App 1",
								},
							},
							Installation: kotsv1beta1.Installation{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "kots.io/v1beta1",
									Kind:       "Installation",
								},
							},
						},
						parentSequence: 1,
					},
				},
				isScheduled: true,
				snapshotTTL: 24 * time.Hour,
				ec: &ecInstanceBackupMetadata{
					installation:         installation,
					seaweedFSS3ServiceIP: "10.96.0.10",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mock_store.NewMockStore(ctrl)
			store.SetStore(mockStore)

			t.Cleanup(func() {
				store.SetStore(nil)
			})

			if tt.setup != nil {
				tt.setup(t, mockStore)
			}

			got, err := getInstanceBackupMetadata(context.Background(), tt.args.k8sClient, tt.args.ctrlClient, tt.args.veleroClient, tt.args.cluster, tt.args.isScheduled)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Regexp(t, "^"+tt.want.backupName, got.backupName)
			assert.NotZero(t, got.backupReqestedAt)
			tt.want.backupName = got.backupName
			tt.want.backupReqestedAt = got.backupReqestedAt

			assert.Equal(t, tt.want, got)
		})
	}
}

func setupArchiveDirectoriesAndFiles(archiveDir string, files map[string]string) error {
	for path, content := range files {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(filepath.Join(archiveDir, dir), 0744); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(archiveDir, path), []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func Test_getBackupNameFromPrefix(t *testing.T) {
	type args struct {
		appSlug string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "basic",
			args: args{
				appSlug: "test",
			},
			want: `^test-[a-f0-9]{8}$`,
		},
		{
			name: "truncate",
			args: args{
				appSlug: "test-truncate-this-string-to-a-valid-backup-name-length",
			},
			want: `^test-truncate-this-string-to-a-valid-backup-name-lengt-[a-f0-9]{8}$`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBackupNameFromPrefix(tt.args.appSlug)
			assert.Regexp(t, tt.want, got)
			assert.LessOrEqual(t, len(got), validation.DNS1035LabelMaxLength)
		})
	}
}
