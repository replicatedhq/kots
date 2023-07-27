package binaries

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_discoverKustomizeVersionsFromDir(t *testing.T) {
	tests := []struct {
		name    string
		binPath string
		bins    []string
		want    []kustomizeFuzzyVersion
		wantErr bool
	}{
		{
			name:    "basic",
			binPath: "/usr/local/bin",
			bins: []string{
				"kustomize3.10.0", "kustomize3", "kustomize4", "kustomize",
				"no-match",
			},
			want: []kustomizeFuzzyVersion{
				newKustomizeFuzzyVersion(4, "/usr/local/bin/kustomize4"),
				newKustomizeFuzzyVersion(3, "/usr/local/bin/kustomize3"),
				newKustomizeFuzzyVersion(0, "/usr/local/bin/kustomize"),
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

			got, err := discoverKustomizeVersionsFromDir(fileSystem, tt.binPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("discoverKustomizeVersionsFromDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("discoverKustomizeVersionsFromDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetKustomizePathForVersion(t *testing.T) {
	knownKustomizeVersions = []kustomizeFuzzyVersion{
		newKustomizeFuzzyVersion(4, "/usr/local/bin/kustomize4"),
		newKustomizeFuzzyVersion(3, "/usr/local/bin/kustomize3"),
		newKustomizeFuzzyVersion(0, "/usr/local/bin/kustomize"),
	}
	sort.Sort(kustomizeVersions(knownKustomizeVersions))
	defer func() {
		knownKustomizeVersions = nil
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
			want:       "/usr/local/bin/kustomize",
		},
		{
			name:       "notexist",
			userString: "1.11.5",
			want:       "/usr/local/bin/kustomize",
		},
		{
			name:       "exact",
			userString: "3.10.0",
			want:       "/usr/local/bin/kustomize3",
		},
		{
			name:       "wrong patch",
			userString: "3.5.4",
			want:       "/usr/local/bin/kustomize3",
		},
		{
			name:       "3",
			userString: "3",
			want:       "/usr/local/bin/kustomize3",
		},
		{
			name:       "3.x.x",
			userString: "3.x.x",
			want:       "/usr/local/bin/kustomize3",
		},
		// this format is not supported
		// {
		// 	name:       "3.10.x",
		// 	userString: "3.10.x",
		// 	want:       "/usr/local/bin/kustomize3",
		// },
		{
			name:       "<4.0.0",
			userString: "<4.0.0",
			want:       "/usr/local/bin/kustomize3",
		},
		{
			name:       ">3.0.0 <=4.0.0",
			userString: ">3.0.0 <=4.0.0",
			want:       "/usr/local/bin/kustomize4",
		},
		{
			name:       "<5.0.0",
			userString: "<5.0.0",
			want:       "/usr/local/bin/kustomize4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetKustomizePathForVersion(tt.userString)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetKustomizePathForVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetKustomizePathForVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
