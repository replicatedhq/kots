package midstream

import (
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
)

type WriteOptions struct {
	MidstreamDir string
	BaseDir      string
}

func (m *Midstream) WriteMidstream(options WriteOptions) error {
	relativeBaseDir, err := filepath.Rel(options.MidstreamDir, options.BaseDir)
	if err != nil {
		return errors.Wrap(err, "failed to determine relative path for base from midstream")
	}

	renderDir := options.MidstreamDir

	_, err = os.Stat(renderDir)
	if err == nil {
		// no error, the midstream already exists
		return nil
	}

	fileRenderPath := path.Join(renderDir, "kustomization.yaml")
	dir, _ := path.Split(fileRenderPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0744); err != nil {
			return errors.Wrap(err, "failed to mkdir")
		}
	}

	m.Kustomization.Bases = []string{
		relativeBaseDir,
	}

	if err := k8sutil.WriteKustomizationToFile(m.Kustomization, fileRenderPath); err != nil {
		return errors.Wrap(err, "failed to write kustomization to file")
	}

	return nil
}
