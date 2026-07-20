package util

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractTGZArchive_overwriteExisting(t *testing.T) {
	dir := t.TempDir()

	writeTar := func() string {
		src, err := os.Create(filepath.Join(dir, "test.tar"))
		require.NoError(t, err)
		defer src.Close()

		gw := gzip.NewWriter(src)
		defer gw.Close()

		tw := tar.NewWriter(gw)
		defer tw.Close()

		file1 := []byte("Hello, World!")
		file2 := []byte("Hello, Another World!")

		err = tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     "test.txt",
			Size:     int64(len(file1)),
			Mode:     0644,
		})
		require.NoError(t, err)
		_, err = tw.Write(file1)
		require.NoError(t, err)

		err = tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     "test.txt",
			Size:     int64(len(file2)),
			Mode:     0644,
		})
		require.NoError(t, err)
		_, err = tw.Write(file2)
		require.NoError(t, err)

		return src.Name()
	}

	src := writeTar()

	dest := filepath.Join(dir, "test")

	err := ExtractTGZArchive(src, dest)
	require.NoError(t, err)

	files, err := os.ReadDir(dest)
	require.NoError(t, err)
	require.Equal(t, 1, len(files))

	content, err := os.ReadFile(filepath.Join(dest, files[0].Name()))
	require.NoError(t, err)
	assert.Equal(t, "Hello, Another World!", string(content))
}

func TestExtractTGZArchive_TarSlip(t *testing.T) {
	dir := t.TempDir()
	destDir := filepath.Join(dir, "dest")

	tests := []struct {
		name      string
		entryName string
		wantErr   bool
		wantFile  string
	}{
		{
			name:      "traversal with dot-dot",
			entryName: "../../pwned",
			wantErr:   true,
		},
		{
			name:      "nested traversal",
			entryName: "foo/../../pwned",
			wantErr:   true,
		},
		{
			name:      "absolute path",
			entryName: "/etc/passwd",
			wantErr:   true,
		},
		{
			name:      "normal file",
			entryName: "app/foo.txt",
			wantErr:   false,
			wantFile:  "app/foo.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, os.RemoveAll(destDir))
			require.NoError(t, os.MkdirAll(destDir, 0755))

			src := filepath.Join(dir, "test.tar.gz")
			f, err := os.Create(src)
			require.NoError(t, err)
			defer f.Close()

			gw := gzip.NewWriter(f)
			tw := tar.NewWriter(gw)
			err = tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeReg,
				Name:     tt.entryName,
				Size:     4,
				Mode:     0644,
			})
			require.NoError(t, err)
			_, err = tw.Write([]byte("data"))
			require.NoError(t, err)
			require.NoError(t, tw.Close())
			require.NoError(t, gw.Close())

			err = ExtractTGZArchive(src, destDir)
			if tt.wantErr {
				require.Error(t, err)
				_, err := os.Stat(filepath.Join(dir, "pwned"))
				require.True(t, os.IsNotExist(err), "traversal file was written outside dest dir")
				return
			}
			require.NoError(t, err)
			content, err := os.ReadFile(filepath.Join(destDir, tt.wantFile))
			require.NoError(t, err)
			assert.Equal(t, "data", string(content))
		})
	}
}
