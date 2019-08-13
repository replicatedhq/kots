package base

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"
)

type WriteOptions struct {
	BaseDir   string
	Overwrite bool
}

func (b *Base) WriteBase(options WriteOptions) error {
	renderDir := options.BaseDir

	_, err := os.Stat(renderDir)
	if err == nil {
		if options.Overwrite {
			if err := os.RemoveAll(renderDir); err != nil {
				return errors.Wrap(err, "failed to remove previous content in base")
			}
		} else {
			return fmt.Errorf("directory %s already exists", renderDir)
		}
	}

	for _, file := range b.Files {
		fileRenderPath := path.Join(renderDir, file.Path)
		d, _ := path.Split(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0744); err != nil {
				return errors.Wrap(err, "failed to mkdir")
			}
		}
		if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
			return errors.Wrap(err, "failed to write base file")
		}
	}

	return nil
}

func (b *Base) GetOverlaysDir(options WriteOptions) string {
	renderDir := options.BaseDir

	return path.Join(renderDir, "..", "overlays")
}
