package version

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/assert"
)

func Test_KotsadmTagForVersionString(t *testing.T) {
	tests := []struct {
		version string
		expect  string
	}{
		{
			version: "v1.13.9-57-g07f06e6-dirty",
			expect:  "alpha",
		},
		{
			version: "v1.14.0-beta",
			expect:  "v1.14.0-beta",
		},
		{
			version: "v1.14.0",
			expect:  "v1.14.0",
		},
		{
			version: "1.14.0",
			expect:  "v1.14.0",
		},
	}

	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			actual := KotsadmTagForVersionString(test.version)
			assert.Equal(t, test.expect, actual)
		})
	}
}

func Test_KotsadmRegistry(t *testing.T) {
	tests := []struct {
		name              string
		overrideVersion   string
		overrideRegistry  string
		overrideNamespace string
		expected          string
	}{
		{
			name:     "no overrides",
			expected: "kotsadm",
		},
		{
			name:             "local registry",
			overrideRegistry: "localhost:32000",
			expected:         "localhost:32000",
		},
		{
			name:              "local registry, custom namespace",
			overrideRegistry:  "registry.somebigbank.com",
			overrideNamespace: "my-namespace/with/multiple/components",
			expected:          "registry.somebigbank.com/my-namespace/with/multiple/components",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := types.KotsadmOptions{
				OverrideVersion:   test.overrideVersion,
				OverrideRegistry:  test.overrideRegistry,
				OverrideNamespace: test.overrideNamespace,
			}

			actual := KotsadmRegistry(options)
			assert.Equal(t, test.expected, actual)
		})
	}
}
