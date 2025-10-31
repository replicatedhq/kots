package lint

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/replicatedhq/kots/pkg/lint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromDirectory(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files
	err := os.WriteFile(filepath.Join(tmpDir, "deployment.yaml"), []byte("apiVersion: v1\nkind: Deployment"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "service.yaml"), []byte("apiVersion: v1\nkind: Service"), 0644)
	require.NoError(t, err)

	// Create a subdirectory with a file
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(subDir, "configmap.yaml"), []byte("apiVersion: v1\nkind: ConfigMap"), 0644)
	require.NoError(t, err)

	// Load files
	files, err := LoadFromDirectory(tmpDir)
	require.NoError(t, err)

	// Verify
	assert.Len(t, files, 3)

	// Check that paths are relative
	for _, file := range files {
		assert.NotContains(t, file.Path, tmpDir)
		assert.True(t, file.IsYAML())
		assert.NotEmpty(t, file.Content)
	}
}

func TestLoadFromTar(t *testing.T) {
	// Create a tar archive in memory
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add files to tar
	files := map[string]string{
		"deployment.yaml": "apiVersion: v1\nkind: Deployment",
		"service.yaml":    "apiVersion: v1\nkind: Service",
	}

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		err := tw.WriteHeader(hdr)
		require.NoError(t, err)

		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}

	err := tw.Close()
	require.NoError(t, err)

	// Load from tar
	specFiles, err := LoadFromTar(&buf)
	require.NoError(t, err)

	// Verify
	assert.Len(t, specFiles, 2)
	for _, file := range specFiles {
		assert.Contains(t, files, file.Path)
		assert.Equal(t, files[file.Path], file.Content)
	}
}

func TestSpecFileIsTarGz(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"chart.tgz", true},
		{"chart.tar.gz", true},
		{"deployment.yaml", false},
		{"file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			file := types.SpecFile{Path: tt.path}
			assert.Equal(t, tt.expected, file.IsTarGz())
		})
	}
}

func TestSpecFileIsYAML(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"deployment.yaml", true},
		{"service.yml", true},
		{"chart.tgz", false},
		{"file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			file := types.SpecFile{Path: tt.path}
			assert.Equal(t, tt.expected, file.IsYAML())
		})
	}
}

func TestSpecFilesUnnest(t *testing.T) {
	files := types.SpecFiles{
		{
			Name: "parent1",
			Path: "parent1.yaml",
			Children: types.SpecFiles{
				{Name: "child1", Path: "child1.yaml"},
				{Name: "child2", Path: "child2.yaml"},
			},
		},
		{
			Name: "parent2",
			Path: "parent2.yaml",
		},
	}

	unnested := files.Unnest()

	assert.Len(t, unnested, 3)
	assert.Equal(t, "child1", unnested[0].Name)
	assert.Equal(t, "child2", unnested[1].Name)
	assert.Equal(t, "parent2", unnested[2].Name)
}

func TestSpecFilesSeparate(t *testing.T) {
	multiDoc := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`

	files := types.SpecFiles{
		{
			Name:    "multi.yaml",
			Path:    "multi.yaml",
			Content: multiDoc,
		},
	}

	separated, err := files.Separate()
	require.NoError(t, err)

	assert.Len(t, separated, 2)
	assert.Equal(t, "multi.yaml", separated[0].Path)
	assert.Equal(t, "multi.yaml", separated[1].Path)
	assert.Equal(t, 0, separated[0].DocIndex)
	assert.Equal(t, 1, separated[1].DocIndex)
	assert.Contains(t, separated[0].Content, "config1")
	assert.Contains(t, separated[1].Content, "config2")
}
