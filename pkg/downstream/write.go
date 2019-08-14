package downstream

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
)

type WriteOptions struct {
	DownstreamDir string
	MidstreamDir  string
	Overwrite     bool
}

func (d *Downstream) WriteDownstream(options WriteOptions) error {
	relativeMidstreamDir, err := filepath.Rel(options.DownstreamDir, options.MidstreamDir)
	if err != nil {
		return errors.Wrap(err, "failed to determine relative path for base from midstream")
	}

	renderDir := options.DownstreamDir

	_, err = os.Stat(renderDir)
	if err == nil {
		if options.Overwrite {
			if err := os.RemoveAll(renderDir); err != nil {
				return errors.Wrap(err, "failed to remove previous content in downstream")
			}
		} else {
			return fmt.Errorf("directory %s already exists", renderDir)
		}
	}

	fileRenderPath := path.Join(renderDir, "kustomization.yaml")
	dir, _ := path.Split(fileRenderPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0744); err != nil {
			return errors.Wrap(err, "failed to mkdir")
		}
	}

	d.Kustomization.Bases = []string{
		relativeMidstreamDir,
	}

	if err := k8sutil.WriteKustomizationToFile(d.Kustomization, fileRenderPath); err != nil {
		return errors.Wrap(err, "failed to write kustomization to file")
	}

	return nil
}
