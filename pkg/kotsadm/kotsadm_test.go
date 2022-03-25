package kotsadm

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_isKotsadmClusterScoped(t *testing.T) {
	tests := []struct {
		name                string
		applicationMetadata []byte
		useMinimalRBAC      bool
		expected            bool
	}{
		{
			name:                "no metadata without override",
			applicationMetadata: nil,
			expected:            true,
		},
		{
			name:                "no metadata with override",
			applicationMetadata: nil,
			useMinimalRBAC:      true,
			expected:            true,
		},
		{
			name: "without additional namespaces",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`,
			),
			expected: true,
		},
		{
			name: "with empty additional namespaces",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  additionalNamespaces: []
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`,
			),
			expected: true,
		},
		{
			name: "with static additional namespaces",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  additionalNamespaces:
    - other1
    - other2
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`,
			),
			expected: true,
		},
		{
			name: "with wildcard namespace",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  additionalNamespaces:
    - "*"
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`,
			),
			expected: true,
		},
		{
			name: "with static and wildcard namespace",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  additionalNamespaces:
    - "*"
    - "test"
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`,
			),
			expected: true,
		},
		{
			name: "with cluster scope requested",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  requireMinimalRBACPrivileges: false
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`),
			expected: true,
		},
		{
			name: "with cluster scope requested plus ignored override",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  requireMinimalRBACPrivileges: false
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`),
			useMinimalRBAC: true,
			expected:       true,
		},
		{
			name: "with cluster scope requested plus override plus enforced override",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  requireMinimalRBACPrivileges: false
  supportMinimalRBACPrivileges: true
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`),
			useMinimalRBAC: true,
			expected:       false,
		},
		{
			name: "with minimal scope requested",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  requireMinimalRBACPrivileges: true
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`),
			expected: false,
		},
		{
			name: "with wildcard namespace",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  requireMinimalRBACPrivileges: true
  additionalNamespaces:
    - "*"
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`,
			),
			expected: true,
		},
		{
			name: "with static and wildcard namespace",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  requireMinimalRBACPrivileges: true
  additionalNamespaces:
    - "*"
    - "test"
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`,
			),
			expected: true,
		},
		{
			name: "with static additional namespaces",
			applicationMetadata: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  requireMinimalRBACPrivileges: true
  additionalNamespaces:
    - other1
    - other2
  icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png`,
			),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			deployOptions := &types.DeployOptions{
				ApplicationMetadata: test.applicationMetadata,
				UseMinimalRBAC:      test.useMinimalRBAC,
			}
			actual, err := isKotsadmClusterScoped(deployOptions)
			req.NoError(err)

			assert.Equal(t, test.expected, actual)
		})
	}
}
