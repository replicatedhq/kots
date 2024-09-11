package downstream

import (
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
)

type WriteOptions struct {
	DownstreamDir string
	MidstreamDir  string
}

func (d *Downstream) WriteDownstream(options WriteOptions) error {
	relativeMidstreamDir, err := filepath.Rel(options.DownstreamDir, options.MidstreamDir)
	if err != nil {
		return errors.Wrap(err, "failed to determine relative path for base from midstream")
	}

	renderDir := options.DownstreamDir

	fileRenderPath := path.Join(renderDir, "kustomization.yaml")

	_, err = os.Stat(fileRenderPath)
	if err == nil {
		// We intentionally don't support overwriting downstreams...  this is user-created content
		// and the user should be intentional about removing it

		// But it's also not an error
		return nil
	}

	dir, _ := path.Split(fileRenderPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0744); err != nil {
			return errors.Wrap(err, "failed to mkdir")
		}
	}

	d.Kustomization.Bases = []string{
		relativeMidstreamDir,
	}

	if err := k8sutil.WriteKustomizationToFile(*d.Kustomization, fileRenderPath); err != nil {
		return errors.Wrap(err, "failed to write kustomization to file")
	}

	return nil
}
