package version

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "empty",
			want: "",
		},
		{
			name: "version string",
			want: "v0.1.2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			version = tt.want
			initBuild()

			got := Version()
			req.Equal(tt.want, got)
		})
	}
}

func TestGitSHA(t *testing.T) {
	tests := []struct {
		name string
		sha  string
		want string
	}{
		{
			name: "empty",
			sha:  "",
			want: "",
		},
		{
			name: "too short",
			sha:  "123456",
			want: "",
		},
		{
			name: "7 chars",
			sha:  "1234567",
			want: "1234567",
		},
		{
			name: "full sha",
			sha:  "e21cf800acca2aa972e7f5f65f7134b5da92f05f",
			want: "e21cf80",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			gitSHA = tt.sha
			initBuild()

			got := GitSHA()
			req.Equal(tt.want, got)
		})
	}
}

func TestBuildTime(t *testing.T) {
	req := require.New(t)
	aTime, err := time.Parse(time.RFC3339, "2019-06-26T18:53:19Z")
	req.NoError(err, "parse constant time")

	tests := []struct {
		name       string
		timestring string
		want       time.Time
	}{
		{
			name:       "empty",
			timestring: "",
			want:       time.Time{},
		},
		{
			name:       "proper format",
			timestring: "2019-06-26T18:53:19Z",
			want:       aTime,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			buildTime = tt.timestring
			initBuild()

			got := BuildTime()
			req.Equal(tt.want, got)
		})
	}
}

func TestGetBuild(t *testing.T) {
	tests := []struct {
		name      string
		gitSHA    string
		version   string
		buildTime string
		want      Build
	}{
		{
			name:   "goInfo",
			gitSHA: "12345678",
			want: Build{
				GitSHA: "1234567",
				GoInfo: getGoInfo(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			version = tt.version
			gitSHA = tt.gitSHA
			buildTime = tt.buildTime

			initBuild()

			got := GetBuild()
			got.RunAt = nil
			req.Equal(tt.want, got)
		})
	}
}
