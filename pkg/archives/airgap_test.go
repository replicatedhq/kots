package archives

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/util"
)

func TestFilterAirgapBundle(t *testing.T) {
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

			got, err := FilterAirgapBundle(airgapBundlePath, tt.filesToInclude)
			if err != nil {
				t.Errorf("FilterAirgapBundle() error = %v", err)
				return
			}

			b, err = os.ReadFile(got)
			if err != nil {
				t.Errorf("failed to read filtered airgap bundle: %v", err)
			}

			gotFiles, err := util.TGZToFiles(b)
			if err != nil {
				t.Errorf("failed to convert filtered airgap bundle to files: %v", err)
			}

			if !reflect.DeepEqual(gotFiles, tt.wantBundleFiles) {
				t.Errorf("FilterAirgapBundle() = %v, want %v", gotFiles, tt.wantBundleFiles)
			}
		})
	}
}
