package archives

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTGZ(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "empty",
			input: "",
			want:  false,
		},
		{
			name:  "not a tgz",
			input: "bm90IGEgdGd6Cg==",
			want:  false,
		},
		{
			name:  "tgz",
			input: "H4sIAE0QXGQAA+2RQQ6DMAwE8xS/oGyCQ94ToaggcQqG9vmNSjmUA1Kl+kTmsrJsyWtvzP0wrqkxigAI3tNbu00Lu26FZXYBFvCBYF1wzpDXNLWzzBJzsdLnON5P5h5DStNJ//so+rNLNeInf0mz3OQpGjvKPzrmX/JnbmEIGmaOXDz/SqVyXV53bklCAAgAAA==",
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			b, err := base64.StdEncoding.DecodeString(tt.input)
			if err != nil {
				t.Errorf("failed to decode input: %v", err)
			}

			if got := IsTGZ(b); got != tt.want {
				t.Errorf("IsTGZ() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractTGZArchiveFromReader_TarSlip(t *testing.T) {
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

			var buf bytes.Buffer
			gw := gzip.NewWriter(&buf)
			tw := tar.NewWriter(gw)
			err := tw.WriteHeader(&tar.Header{
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

			err = ExtractTGZArchiveFromReader(bytes.NewReader(buf.Bytes()), destDir)
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
