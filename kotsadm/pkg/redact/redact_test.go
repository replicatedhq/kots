package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
)

func Test_getSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "all alphanumeric",
			input: "aBC123",
			want:  "aBC123",
		},
		{
			name:  "dashes",
			input: "abc-123",
			want:  "abc-123",
		},
		{
			name:  "spaces",
			input: "abc 123",
			want:  "abc-123",
		},
		{
			name:  "aymbols",
			input: "abc%^123!@#",
			want:  "abc123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			req.Equal(tt.want, getSlug(tt.input))
		})
	}
}
