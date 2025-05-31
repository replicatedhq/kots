package archiveutil

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractTGZ_overwriteExisting(t *testing.T) {
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

	err := ExtractTGZ(t.Context(), src, dest)
	require.NoError(t, err)

	files, err := os.ReadDir(dest)
	require.NoError(t, err)
	require.Equal(t, 1, len(files))

	content, err := os.ReadFile(filepath.Join(dest, files[0].Name()))
	require.NoError(t, err)
	assert.Equal(t, "Hello, Another World!", string(content))
}
