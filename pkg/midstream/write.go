package midstream

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
)

type WriteOptions struct {
	MidstreamDir string
	BaseDir      string
	Overwrite    bool
}

func (m *Midstream) WriteMidstream(options WriteOptions) error {
	relativeBaseDir, err := filepath.Rel(options.MidstreamDir, options.BaseDir)
	if err != nil {
		return errors.Wrap(err, "failed to determine relative path for base from midstream")
	}

	renderDir := options.MidstreamDir

	_, err = os.Stat(renderDir)
	if err == nil {
		if options.Overwrite {
			if err := os.RemoveAll(renderDir); err != nil {
				return errors.Wrap(err, "failed to remove previous content in midstream")
			}
		} else {
			return fmt.Errorf("directory %s already exists", renderDir)
		}
	}

	fileRenderPath := path.Join(renderDir, "kustomization.yaml")
	d, _ := path.Split(fileRenderPath)
	if _, err := os.Stat(d); os.IsNotExist(err) {
		if err := os.MkdirAll(d, 0744); err != nil {
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
