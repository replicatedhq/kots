package binaries

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_discoverKubectlVersionsFromDir(t *testing.T) {
	tests := []struct {
		name    string
		binPath string
		bins    []string
		want    []kubectlFuzzyVersion
		wantErr bool
	}{
		{
			name:    "basic",
			binPath: "/usr/local/bin",
			bins: []string{
				"kubectl-v1.14", "kubectl-v1.16", "kubectl-v1.17", "kubectl-v1.18", "kubectl-v1.19", "kubectl-v1.20", "kubectl-v1.21", "kubectl",
				"no-match",
			},
			want: []kubectlFuzzyVersion{
				newKubectlFuzzyVersion(1, 21, "/usr/local/bin/kubectl-v1.21"),
				newKubectlFuzzyVersion(1, 20, "/usr/local/bin/kubectl-v1.20"),
				newKubectlFuzzyVersion(1, 19, "/usr/local/bin/kubectl-v1.19"),
				newKubectlFuzzyVersion(1, 18, "/usr/local/bin/kubectl-v1.18"),
				newKubectlFuzzyVersion(1, 17, "/usr/local/bin/kubectl-v1.17"),
				newKubectlFuzzyVersion(1, 16, "/usr/local/bin/kubectl-v1.16"),
				newKubectlFuzzyVersion(1, 14, "/usr/local/bin/kubectl-v1.14"),
				newKubectlFuzzyVersion(0, 0, "/usr/local/bin/kubectl"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirPath := t.TempDir()

			binPath := filepath.Join(dirPath, tt.binPath)
			require.NoError(t, os.MkdirAll(binPath, 0755))
			for _, bin := range tt.bins {
				require.NoError(t, os.WriteFile(filepath.Join(binPath, bin), nil, 0755))
			}

			fileSystem := os.DirFS(dirPath)

			got, err := discoverKubectlVersionsFromDir(fileSystem, tt.binPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("discoverKubectlVersionsFromDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("discoverKubectlVersionsFromDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetKubectlPathForVersion(t *testing.T) {
	knownKubectlVersions = []kubectlFuzzyVersion{
		newKubectlFuzzyVersion(1, 21, "/usr/local/bin/kubectl-v1.21"),
		newKubectlFuzzyVersion(1, 20, "/usr/local/bin/kubectl-v1.20"),
		newKubectlFuzzyVersion(1, 19, "/usr/local/bin/kubectl-v1.19"),
		newKubectlFuzzyVersion(1, 18, "/usr/local/bin/kubectl-v1.18"),
		newKubectlFuzzyVersion(1, 17, "/usr/local/bin/kubectl-v1.17"),
		newKubectlFuzzyVersion(1, 16, "/usr/local/bin/kubectl-v1.16"),
		newKubectlFuzzyVersion(1, 14, "/usr/local/bin/kubectl-v1.14"),
		newKubectlFuzzyVersion(0, 0, "/usr/local/bin/kubectl"),
	}
	sort.Sort(kubectlVersions(knownKubectlVersions))
	defer func() {
		knownKubectlVersions = nil
	}()

	tests := []struct {
		name       string
		userString string
		want       string
		wantErr    bool
	}{
		{
			name:       "1",
			userString: "1",
			want:       "/usr/local/bin/kubectl",
		},
		{
			name:       "notexist",
			userString: "1.11.5",
			want:       "/usr/local/bin/kubectl",
		},
		{
			name:       "exact",
			userString: "1.16.15",
			want:       "/usr/local/bin/kubectl-v1.16",
		},
		{
			name:       "wrong patch",
			userString: "1.16.3",
			want:       "/usr/local/bin/kubectl-v1.16",
		},
		{
			name:       "1.14.x",
			userString: "1.14.x",
			want:       "/usr/local/bin/kubectl-v1.14",
		},
		{
			name:       "<1.15.0",
			userString: "<1.15.0",
			want:       "/usr/local/bin/kubectl-v1.14",
		},
		{
			name:       ">1.15.0 <1.17.0",
			userString: ">1.15.0 <1.17.0",
			want:       "/usr/local/bin/kubectl-v1.16",
		},
		{
			name:       "<1.17.0",
			userString: "<1.17.0",
			want:       "/usr/local/bin/kubectl-v1.16",
		},
		{
			name:       "<=1.17.0",
			userString: "<=1.17.0",
			want:       "/usr/local/bin/kubectl-v1.17",
		},
		{
			name:       "1.17",
			userString: "1.17",
			want:       "/usr/local/bin/kubectl-v1.17",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetKubectlPathForVersion(tt.userString)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetKubectlPathForVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetKubectlPathForVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
