package pull

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRewriteUpstream(t *testing.T) {
	tests := []struct {
		upstreamURI string
		expected    string
	}{
		{
			upstreamURI: "app-slug",
			expected:    "replicated://app-slug",
		},
		{
			upstreamURI: "app-slug/beta",
			expected:    "replicated://app-slug/beta",
		},
		{
			upstreamURI: "helm://stable/mysql",
			expected:    "helm://stable/mysql",
		},
	}
	for _, test := range tests {
		t.Run(test.upstreamURI, func(t *testing.T) {
			actual := RewriteUpstream(test.upstreamURI)
			assert.Equal(t, actual, test.expected)
		})
	}
}
