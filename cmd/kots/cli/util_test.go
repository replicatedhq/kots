package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/replicatedhq/kots/pkg/util"
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
			want:  util.HomeDir(),
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

func Test_getHostFromEndpoint(t *testing.T) {
	type args struct {
		endpoint string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"endpoint without scheme",
			args{
				endpoint: "localhost",
			},
			"localhost",
			false,
		},
		{
			"endpoint with scheme",
			args{
				endpoint: "https://localhost",
			},
			"localhost",
			false,
		},
		{
			"endpoint with port",
			args{
				endpoint: "localhost:3000",
			},
			"localhost:3000",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getHostFromEndpoint(tt.args.endpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("getHostFromEndpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getHostFromEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}
