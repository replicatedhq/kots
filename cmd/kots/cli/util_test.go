package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandDir(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already abs",
			input: "/var/lib",
			want:  "/var/lib",
		},
		{
			name:  "home",
			input: "~",
			want:  homeDir(),
		},
		{
			name:  "./cmd",
			input: "./cmd",
			want:  filepath.Join(wd, "cmd"),
		},
		{
			name:  "empty string should remain empty",
			input: "",
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got := ExpandDir(tt.input)
			req.Equal(tt.want, got)
		})
	}
}
