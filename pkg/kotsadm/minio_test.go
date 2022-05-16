package kotsadm

import (
	"testing"

	"github.com/stretchr/testify/require"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func Test_cleanUpMigrationArtifact(t *testing.T) {
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

			clientset := testclient.NewSimpleClientset()

			if test.hasArtifact {
				err := createMigrationArtifact(clientset, test.namespace)
				req.NoError(err)
			}

			err := cleanUpMigrationArtifact(clientset, test.namespace)
			req.NoError(err)
		})
	}
}
