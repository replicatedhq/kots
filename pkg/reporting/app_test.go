package reporting

import (
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/store"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	velerofake "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/fake"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_getSnapshotReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	t.Setenv("POD_NAMESPACE", "default")

	testVeleroNamespace := "velero"
	veleroNamespaceConfigmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kotsadm-velero-namespace",
		},
		Data: map[string]string{
			"veleroNamespace": testVeleroNamespace,
		},
	}
	veleroDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "velero",
			Namespace: testVeleroNamespace,
		},
	}

	testAppID := "test-app-id"
	testClusterID := "test-cluster-id"

	type args struct {
		kotsStore    store.Store
		clientset    kubernetes.Interface
		veleroClient veleroclientv1.VeleroV1Interface
		appID        string
		clusterID    string
	}
	tests := []struct {
		name                  string
		args                  args
		mockStoreExpectations func()
		want                  *SnapshotReport
		wantErr               bool
	}{
		{
			name: "happy path with schedule and ttl",
			args: args{
				kotsStore: mockStore,
				clientset: fake.NewSimpleClientset(veleroNamespaceConfigmap, veleroDeployment),
				veleroClient: velerofake.NewSimpleClientset(
					backupStorageLocationWithProvider(testVeleroNamespace, "aws"),
				).VeleroV1(),
				appID:     testAppID,
				clusterID: testClusterID,
			},
			mockStoreExpectations: func() {
				mockStore.EXPECT().ListClusters().Return([]*downstreamtypes.Downstream{
					{
						ClusterID:        testClusterID,
						SnapshotSchedule: "0 0 * * *",
						SnapshotTTL:      "720h",
					},
				}, nil)
				mockStore.EXPECT().GetApp(testAppID).Return(&apptypes.App{
					ID:               testAppID,
					SnapshotSchedule: "0 0 * * MON",
					SnapshotTTL:      "168h",
				}, nil)
			},
			want: &SnapshotReport{
				Provider:        "aws",
				FullSchedule:    "0 0 * * *",
				FullTTL:         "720h",
				PartialSchedule: "0 0 * * MON",
				PartialTTL:      "168h",
			},
		},
		{
			name: "happy path with default ttl only",
			args: args{
				kotsStore: mockStore,
				clientset: fake.NewSimpleClientset(veleroNamespaceConfigmap, veleroDeployment),
				veleroClient: velerofake.NewSimpleClientset(
					backupStorageLocationWithProvider(testVeleroNamespace, "aws"),
				).VeleroV1(),
				appID:     testAppID,
				clusterID: testClusterID,
			},
			mockStoreExpectations: func() {
				mockStore.EXPECT().ListClusters().Return([]*downstreamtypes.Downstream{
					{
						ClusterID:        testClusterID,
						SnapshotSchedule: "",
						SnapshotTTL:      "720h",
					},
				}, nil)
				mockStore.EXPECT().GetApp(testAppID).Return(&apptypes.App{
					ID:               testAppID,
					SnapshotSchedule: "",
					SnapshotTTL:      "720h",
				}, nil)
			},
			want: &SnapshotReport{
				Provider:        "aws",
				FullSchedule:    "",
				FullTTL:         "720h",
				PartialSchedule: "",
				PartialTTL:      "720h",
			},
		},
		{
			name: "no backup storage location",
			args: args{
				kotsStore:    mockStore,
				clientset:    fake.NewSimpleClientset(veleroNamespaceConfigmap, veleroDeployment),
				veleroClient: velerofake.NewSimpleClientset().VeleroV1(),
				appID:        testAppID,
				clusterID:    testClusterID,
			},
			mockStoreExpectations: func() {},
			wantErr:               true,
		},
		{
			name: "failed to list clusters",
			args: args{
				kotsStore: mockStore,
				clientset: fake.NewSimpleClientset(veleroNamespaceConfigmap, veleroDeployment),
				veleroClient: velerofake.NewSimpleClientset(
					backupStorageLocationWithProvider(testVeleroNamespace, "aws"),
				).VeleroV1(),
				appID:     testAppID,
				clusterID: testClusterID,
			},
			mockStoreExpectations: func() {
				mockStore.EXPECT().ListClusters().Return(nil, errors.New("failed to list clusters"))
			},
			wantErr: true,
		},
		{
			name: "failed to get app",
			args: args{
				kotsStore: mockStore,
				clientset: fake.NewSimpleClientset(veleroNamespaceConfigmap, veleroDeployment),
				veleroClient: velerofake.NewSimpleClientset(
					backupStorageLocationWithProvider(testVeleroNamespace, "aws"),
				).VeleroV1(),
				appID:     testAppID,
				clusterID: testClusterID,
			},
			mockStoreExpectations: func() {
				mockStore.EXPECT().ListClusters().Return([]*downstreamtypes.Downstream{
					{
						ClusterID:        testClusterID,
						SnapshotSchedule: "0 0 * * *",
						SnapshotTTL:      "720h",
					},
				}, nil)
				mockStore.EXPECT().GetApp(testAppID).Return(nil, errors.New("failed to get app"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockStoreExpectations()
			got, err := getSnapshotReport(tt.args.kotsStore, tt.args.clientset, tt.args.veleroClient, tt.args.appID, tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSnapshotReport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getSnapshotReport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func backupStorageLocationWithProvider(namespace string, provider string) *velerov1.BackupStorageLocation {
	return &velerov1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: namespace,
		},
		Spec: velerov1.BackupStorageLocationSpec{
			Provider: provider,
			Default:  true,
		},
	}
}
