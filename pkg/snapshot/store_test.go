package snapshot

import (
	"context"
	"testing"

	"github.com/replicatedhq/kots/pkg/snapshot/types"
	"github.com/stretchr/testify/require"
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
