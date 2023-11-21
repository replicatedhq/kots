package snapshot

import (
	"context"
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/snapshot/types"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	velerofake "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/fake"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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
		name              string
		args              updateExistingStoreArgs
		wantStore         *types.Store
		wantRestart       bool
		wantErr           bool
		wantValidationErr error
	}{
		{
			name: "update existing store to aws -- should succeed",
			args: updateExistingStoreArgs{
				context:       context.TODO(),
				clientset:     testclient.NewSimpleClientset(),
				existingStore: &types.Store{},
				options: ConfigureStoreOptions{
					Provider: "aws",
					Bucket:   "my-bucket",
					Path:     "my-path",
					AWS: &types.StoreAWS{
						Region:          "us-east-1",
						UseInstanceRole: false,
						AccessKeyID:     "access-key",
						SecretAccessKey: "secret-key",
					},
				},
			},
			wantStore: &types.Store{
				Provider: "aws",
				Bucket:   "my-bucket",
				Path:     "my-path",
				AWS: &types.StoreAWS{
					Region:          "us-east-1",
					UseInstanceRole: false,
					AccessKeyID:     "access-key",
					SecretAccessKey: "secret-key",
				},
			},
			wantRestart: true,
			wantErr:     false,
		},
		{
			name: "update existing store to aws -- using instance role, so credentials are not required and will be set to empty strings",
			args: updateExistingStoreArgs{
				context:       context.TODO(),
				clientset:     testclient.NewSimpleClientset(),
				existingStore: &types.Store{},
				options: ConfigureStoreOptions{
					Provider: "aws",
					Bucket:   "my-bucket",
					Path:     "my-path",
					AWS: &types.StoreAWS{
						Region:          "us-east-1",
						UseInstanceRole: true,
					},
				},
			},
			wantStore: &types.Store{
				Provider: "aws",
				Bucket:   "my-bucket",
				Path:     "my-path",
				AWS: &types.StoreAWS{
					Region:          "us-east-1",
					UseInstanceRole: true,
					AccessKeyID:     "",
					SecretAccessKey: "",
				},
			},
			wantRestart: true,
			wantErr:     false,
		},
		{
			name: "update existing store to aws -- no credentials and not using instance role, should fail with validation error",
			args: updateExistingStoreArgs{
				context:       context.TODO(),
				clientset:     testclient.NewSimpleClientset(),
				existingStore: &types.Store{},
				options: ConfigureStoreOptions{
					Provider: "aws",
					Bucket:   "my-bucket",
					Path:     "my-path",
					AWS: &types.StoreAWS{
						Region:          "us-east-1",
						UseInstanceRole: false,
					},
				},
			},
			wantStore:         nil,
			wantRestart:       false,
			wantErr:           true,
			wantValidationErr: &InvalidStoreDataError{Message: "missing access key id and/or secret access key and/or region"},
		},
		{
			name: "update existing store to aws -- redacted secret key, should fail with validation error",
			args: updateExistingStoreArgs{
				context:       context.TODO(),
				clientset:     testclient.NewSimpleClientset(),
				existingStore: &types.Store{},
				options: ConfigureStoreOptions{
					Provider: "aws",
					Bucket:   "my-bucket",
					Path:     "my-path",
					AWS: &types.StoreAWS{
						Region:          "us-east-1",
						UseInstanceRole: false,
						AccessKeyID:     "access-key",
						SecretAccessKey: "****REDACTED****",
					},
				},
			},
			wantStore:         nil,
			wantRestart:       false,
			wantErr:           true,
			wantValidationErr: &InvalidStoreDataError{Message: "invalid aws secret access key"},
		},
		{
			name: "update existing store to google -- use json file, should succeed",
			args: updateExistingStoreArgs{
				context:       context.TODO(),
				clientset:     testclient.NewSimpleClientset(),
				existingStore: &types.Store{},
				options: ConfigureStoreOptions{
					Provider: "gcp",
					Bucket:   "my-bucket",
					Path:     "my-path",
					Google: &types.StoreGoogle{
						JSONFile: "my-json-file",
					},
				},
			},
			wantStore: &types.Store{
				Provider: "gcp",
				Bucket:   "my-bucket",
				Path:     "my-path",
				Google: &types.StoreGoogle{
					JSONFile: "my-json-file",
				},
			},
			wantRestart: true,
			wantErr:     false,
		},
		{
			name: "update existing store to google -- use instance role and service account, should succeed",
			args: updateExistingStoreArgs{
				context:       context.TODO(),
				clientset:     testclient.NewSimpleClientset(),
				existingStore: &types.Store{},
				options: ConfigureStoreOptions{
					Provider: "gcp",
					Bucket:   "my-bucket",
					Path:     "my-path",
					Google: &types.StoreGoogle{
						UseInstanceRole: true,
						ServiceAccount:  "my-service-account",
					},
				},
			},
			wantStore: &types.Store{
				Provider: "gcp",
				Bucket:   "my-bucket",
				Path:     "my-path",
				Google: &types.StoreGoogle{
					UseInstanceRole: true,
					ServiceAccount:  "my-service-account",
				},
			},
			wantRestart: true,
			wantErr:     false,
		},
		{
			name: "update existing store to google -- no json file, should fail with validation error",
			args: updateExistingStoreArgs{
				context:       context.TODO(),
				clientset:     testclient.NewSimpleClientset(),
				existingStore: &types.Store{},
				options: ConfigureStoreOptions{
					Provider: "gcp",
					Bucket:   "my-bucket",
					Path:     "my-path",
					Google:   &types.StoreGoogle{},
				},
			},
			wantStore:         nil,
			wantRestart:       false,
			wantErr:           true,
			wantValidationErr: &InvalidStoreDataError{Message: "missing JSON file"},
		},
		{
			name: "update existing store to google -- instance role without service account, should fail with validation error",
			args: updateExistingStoreArgs{
				context:       context.TODO(),
				clientset:     testclient.NewSimpleClientset(),
				existingStore: &types.Store{},
				options: ConfigureStoreOptions{
					Provider: "gcp",
					Bucket:   "my-bucket",
					Path:     "my-path",
					Google: &types.StoreGoogle{
						UseInstanceRole: true,
					},
				},
			},
			wantStore:         nil,
			wantRestart:       false,
			wantErr:           true,
			wantValidationErr: &InvalidStoreDataError{Message: "missing service account"},
		},
		{
			name: "update existing store to google -- redacted json file, should fail with validation error",
			args: updateExistingStoreArgs{
				context:       context.TODO(),
				clientset:     testclient.NewSimpleClientset(),
				existingStore: &types.Store{},
				options: ConfigureStoreOptions{
					Provider: "gcp",
					Bucket:   "my-bucket",
					Path:     "my-path",
					Google: &types.StoreGoogle{
						JSONFile: "****REDACTED****",
					},
				},
			},
			wantStore:         nil,
			wantRestart:       false,
			wantErr:           true,
			wantValidationErr: &InvalidStoreDataError{Message: "invalid JSON file"},
		},
		{
			name: "update existing store to azure -- should succeed",
			args: updateExistingStoreArgs{
				context:       context.TODO(),
				clientset:     testclient.NewSimpleClientset(),
				existingStore: &types.Store{},
				options: ConfigureStoreOptions{
					Provider: "azure",
					Bucket:   "my-bucket",
					Path:     "my-path",
					Azure: &types.StoreAzure{
						ResourceGroup:  "my-resource-group",
						StorageAccount: "my-storage-account",
						SubscriptionID: "my-subscription-id",
						TenantID:       "my-tenant-id",
						ClientID:       "my-client-id",
						ClientSecret:   "my-client-secret",
						CloudName:      "my-cloud-name",
					},
				},
			},
			wantStore: &types.Store{
				Provider: "azure",
				Bucket:   "my-bucket",
				Path:     "my-path",
				Azure: &types.StoreAzure{
					ResourceGroup:  "my-resource-group",
					StorageAccount: "my-storage-account",
					SubscriptionID: "my-subscription-id",
					TenantID:       "my-tenant-id",
					ClientID:       "my-client-id",
					ClientSecret:   "my-client-secret",
					CloudName:      "my-cloud-name",
				},
			},
			wantRestart: true,
			wantErr:     false,
		},
		{
			name: "update existing store to azure -- client secret redacted, should fail with validation error",
			args: updateExistingStoreArgs{
				context:       context.TODO(),
				clientset:     testclient.NewSimpleClientset(),
				existingStore: &types.Store{},
				options: ConfigureStoreOptions{
					Provider: "azure",
					Bucket:   "my-bucket",
					Path:     "my-path",
					Azure: &types.StoreAzure{
						ResourceGroup:  "my-resource-group",
						StorageAccount: "my-storage-account",
						SubscriptionID: "my-subscription-id",
						TenantID:       "my-tenant-id",
						ClientID:       "my-client-id",
						ClientSecret:   "****REDACTED****",
						CloudName:      "my-cloud-name",
					},
				},
			},
			wantStore:         nil,
			wantRestart:       false,
			wantErr:           true,
			wantValidationErr: &InvalidStoreDataError{Message: "invalid client secret"},
		},
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

			newStore, needsVeleroRestart, err := buildNewStore(test.args.context, test.args.clientset, test.args.existingStore, test.args.options)
			if test.wantErr {
				req.Error(err)
				if test.wantValidationErr != nil {
					req.Equal(test.wantValidationErr, err)
				}
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

func TestFindBackupStoreLocation(t *testing.T) {

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

	type args struct {
		clientset        kubernetes.Interface
		veleroClient     veleroclientv1.VeleroV1Interface
		kotsadmNamespace string
	}
	tests := []struct {
		name    string
		args    args
		want    *velerov1.BackupStorageLocation
		wantErr bool
	}{
		{
			name: "backup store location found",
			args: args{
				clientset:        fake.NewSimpleClientset(veleroNamespaceConfigmap, veleroDeployment),
				veleroClient:     velerofake.NewSimpleClientset(testBsl).VeleroV1(),
				kotsadmNamespace: "default",
			},
			want: testBsl,
		},
		{
			name: "return nil if no backup store location found",
			args: args{
				clientset:        fake.NewSimpleClientset(veleroNamespaceConfigmap, veleroDeployment),
				veleroClient:     velerofake.NewSimpleClientset().VeleroV1(),
				kotsadmNamespace: "default",
			},
			want: nil,
		},
		{
			name: "return nil if no velero deployment found",
			args: args{
				clientset:        fake.NewSimpleClientset(veleroNamespaceConfigmap),
				veleroClient:     velerofake.NewSimpleClientset(testBsl).VeleroV1(),
				kotsadmNamespace: "default",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := FindBackupStoreLocation(ctx, tt.args.clientset, tt.args.veleroClient, tt.args.kotsadmNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindBackupStoreLocation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindBackupStoreLocation() = %v, want %v", got, tt.want)
			}
		})
	}
}
