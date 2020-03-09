package kotsadm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_isOperatorClusterScoped(t *testing.T) {
	tests := []struct {
		name                string
		applicationMetadata []byte
		expected            bool
	}{
		{
			name:                "no metadata",
			applicationMetadata: nil,
			expected:            false,
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
			expected: false,
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
			expected: false,
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := isOperatorClusterScoped(test.applicationMetadata)
			req.NoError(err)

			assert.Equal(t, test.expected, actual)
		})
	}
}
