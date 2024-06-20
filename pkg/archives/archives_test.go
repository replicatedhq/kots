package archives

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/util"
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

func TestCreateFilteredAirgapBundle(t *testing.T) {
	tests := []struct {
		name            string
		bundleFiles     map[string]string
		filesToInclude  []string
		wantBundleFiles map[string]string
	}{
		{
			name: "slim airgap bundle",
			bundleFiles: map[string]string{
				"airgap.yaml":                     "airgap-metadata",
				"app.tar.gz":                      "application-archive",
				"embedded-cluster/artifacts/kots": "kots-binary",
				"images":                          "image-data",
			},
			filesToInclude: []string{
				"airgap.yaml",
				"app.tar.gz",
				"embedded-cluster/artifacts/kots",
			},
			wantBundleFiles: map[string]string{
				"airgap.yaml":                     "airgap-metadata",
				"app.tar.gz":                      "application-archive",
				"embedded-cluster/artifacts/kots": "kots-binary",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := util.FilesToTGZ(tt.bundleFiles)
			if err != nil {
				t.Errorf("failed to create tgz: %v", err)
			}

			airgapBundlePath := filepath.Join(t.TempDir(), "application.airgap")
			tmpFile, err := os.Create(airgapBundlePath)
			if err != nil {
				t.Errorf("failed to create tmp file: %v", err)
			}

			_, err = tmpFile.Write(b)
			if err != nil {
				t.Errorf("failed to write to tmp file: %v", err)
			}

			got, err := CreateFilteredAirgapBundle(airgapBundlePath, tt.filesToInclude)
			if err != nil {
				t.Errorf("CreateFilteredAirgapBundle() error = %v", err)
				return
			}

			b, err = io.ReadAll(got)
			if err != nil {
				t.Errorf("failed to read filtered airgap bundle: %v", err)
			}

			gotFiles, err := util.TGZToFiles(b)
			if err != nil {
				t.Errorf("failed to convert filtered airgap bundle to files: %v", err)
			}

			if !reflect.DeepEqual(gotFiles, tt.wantBundleFiles) {
				t.Errorf("CreateFilteredAirgapBundle() = %v, want %v", gotFiles, tt.wantBundleFiles)
			}
		})
	}
}
