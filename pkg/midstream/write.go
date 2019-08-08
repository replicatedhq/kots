package midstream

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"
)

type WriteOptions struct {
	MidstreamDir string
	Overwrite    bool
}

func (m *Midstream) WriteMidstream(options WriteOptions) error {
	renderDir := options.MidstreamDir

	_, err := os.Stat(renderDir)
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
	if err := ioutil.WriteFile(fileRenderPath, []byte(m.Kustomization), 0644); err != nil {
		return errors.Wrap(err, "failed to write midstream file")
	}

	return nil
}
