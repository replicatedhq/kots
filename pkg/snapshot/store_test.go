package snapshot

import (
	"context"
	"testing"

	"github.com/replicatedhq/kots/pkg/snapshot/types"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"
)

func Test_updateExistingStore(t *testing.T) {
	type updateExistingStoreArgs struct {
		context       context.Context
		clientset     kubernetes.Interface
		existingStore *types.Store
		options       ConfigureStoreOptions
	}

	hostPathConfig := &types.FileSystemConfig{
		HostPath: pointer.String("/my/host/path"),
	}

	hostPathBucket, err := GetLvpBucket(hostPathConfig)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        updateExistingStoreArgs
		wantStore   *types.Store
		wantRestart bool
		wantErr     bool
	}{
		{
			name: "update existing filesystem store with lvp -- the bucket changes, so we should not restart velero since the plugin will handle it",
			args: updateExistingStoreArgs{
				context:   context.TODO(),
				clientset: testclient.NewSimpleClientset(),
				existingStore: &types.Store{
					Bucket: "old-bucket",
				},
				options: ConfigureStoreOptions{
					FileSystem:      hostPathConfig,
					IsMinioDisabled: true,
				},
			},
			wantStore: &types.Store{
				Bucket:   hostPathBucket,
				Provider: SnapshotStoreHostPathProvider,
				FileSystem: &types.StoreFileSystem{
					Config: hostPathConfig,
				},
			},
			wantRestart: false,
			wantErr:     false,
		},
		{
			name: "update existing filesystem store with lvp -- the bucket is the same, so we should restart velero",
			args: updateExistingStoreArgs{
				context:   context.TODO(),
				clientset: testclient.NewSimpleClientset(),
				existingStore: &types.Store{
					Bucket: hostPathBucket,
				},
				options: ConfigureStoreOptions{
					FileSystem:      hostPathConfig,
					IsMinioDisabled: true,
				},
			},
			wantStore: &types.Store{
				Bucket:   hostPathBucket,
				Provider: SnapshotStoreHostPathProvider,
				FileSystem: &types.StoreFileSystem{
					Config: hostPathConfig,
				},
			},
			wantRestart: true,
			wantErr:     false,
		},
		{
			name: "update existing filesystem store with lvp -- it's a minio migrated store, so it should add the /velero path",
			args: updateExistingStoreArgs{
				context: context.TODO(),
				clientset: testclient.NewSimpleClientset(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      SnapshotMigrationArtifactName,
						Namespace: "default",
					},
				}),
				existingStore: &types.Store{
					Bucket: "old-bucket",
				},
				options: ConfigureStoreOptions{
					FileSystem:       hostPathConfig,
					IsMinioDisabled:  true,
					KotsadmNamespace: "default",
				},
			},
			wantStore: &types.Store{
				Bucket:   hostPathBucket,
				Provider: SnapshotStoreHostPathProvider,
				Path:     "/velero",
				FileSystem: &types.StoreFileSystem{
					Config: hostPathConfig,
				},
			},
			wantRestart: false,
			wantErr:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			newStore, needsVeleroRestart, err := updateExistingStore(test.args.context, test.args.clientset, test.args.existingStore, test.args.options)
			if test.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
			req.Equal(test.wantStore, newStore)
			req.Equal(test.wantRestart, needsVeleroRestart)
		})
	}
}

func Test_validateInternalStore(t *testing.T) {
	req := require.New(t)

	namespace := "default"

	tests := []struct {
		Name         string
		Store        types.Store
		Options      ValidateStoreOptions
		ExpectErrMsg string
	}{
		{
			Name: "valid internal store using pvc",
			Store: types.Store{
				Provider: SnapshotStorePVCProvider,
				Bucket:   SnapshotStorePVCBucket,
				Path:     "",
				Internal: &types.StoreInternal{},
			},
			Options:      ValidateStoreOptions{KotsadmNamespace: namespace},
			ExpectErrMsg: "",
		},
		{
			Name: "invalid internal store using s3",
			Store: types.Store{
				Provider: "aws",
				Bucket:   "snapshot-bucket",
				Path:     "",
				Internal: &types.StoreInternal{
					AccessKeyID:     "access-key",
					SecretAccessKey: "secret-key",
					Endpoint:        "does.not.exist",
					Region:          "us-east-1",
				},
			},
			Options:      ValidateStoreOptions{KotsadmNamespace: "default"},
			ExpectErrMsg: "bucket does not exist",
		},
	}

	for _, test := range tests {
		err := validateStore(context.TODO(), &test.Store, test.Options)
		if test.ExpectErrMsg != "" {
			req.Contains(err.Error(), test.ExpectErrMsg)
		} else {
			req.NoError(err)
		}
	}
}

func Test_isMinioMigration(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		hasArtifact bool
	}{
		{
			name:        "has artifact",
			namespace:   "default",
			hasArtifact: true,
		},
		{
			name:        "no artifact",
			namespace:   "default",
			hasArtifact: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			var clientset *testclient.Clientset
			if test.hasArtifact {
				clientset = testclient.NewSimpleClientset(&corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "ConfigMap",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      SnapshotMigrationArtifactName,
						Namespace: test.namespace,
					},
				})
			} else {
				clientset = testclient.NewSimpleClientset()
			}

			result := isMinioMigration(clientset, test.namespace)
			req.Equal(test.hasArtifact, result)
		})
	}
}
