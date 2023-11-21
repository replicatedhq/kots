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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_getSnapshotReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	t.Setenv("POD_NAMESPACE", "default")

	testVeleroNamespace := "velero"
	testBsl := &velerov1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: testVeleroNamespace,
		},
		Spec: velerov1.BackupStorageLocationSpec{
			Provider: "aws",
			Default:  true,
		},
	}

	testAppID := "test-app-id"
	testClusterID := "test-cluster-id"

	type args struct {
		kotsStore store.Store
		bsl       *velerov1.BackupStorageLocation
		appID     string
		clusterID string
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
				bsl:       testBsl,
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
				bsl:       testBsl,
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
				kotsStore: mockStore,
				bsl:       nil,
				appID:     testAppID,
				clusterID: testClusterID,
			},
			mockStoreExpectations: func() {},
			wantErr:               true,
		},
		{
			name: "failed to list clusters",
			args: args{
				kotsStore: mockStore,
				bsl:       testBsl,
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
				bsl:       testBsl,
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
			got, err := getSnapshotReport(tt.args.kotsStore, tt.args.bsl, tt.args.appID, tt.args.clusterID)
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
