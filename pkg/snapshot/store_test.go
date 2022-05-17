package snapshot

import (
	"context"
	"testing"

	"github.com/replicatedhq/kots/pkg/snapshot/types"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

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
