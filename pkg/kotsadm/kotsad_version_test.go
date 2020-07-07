package kotsadm

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/assert"
	"go.undefinedlabs.com/scopeagent"
)

func Test_kotsadmRegistry(t *testing.T) {
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
			expected:         "localhost:32000/kotsadm",
		},
		{
			name:              "local registry, custom namespace",
			overrideRegistry:  "registry.somebigbank.com",
			overrideNamespace: "my-namespace",
			expected:          "registry.somebigbank.com/my-namespace",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()

			options := types.KotsadmOptions{
				OverrideVersion:   test.overrideVersion,
				OverrideRegistry:  test.overrideRegistry,
				OverrideNamespace: test.overrideNamespace,
			}

			actual := kotsadmRegistry(options)
			assert.Equal(t, test.expected, actual)
		})
	}
}
