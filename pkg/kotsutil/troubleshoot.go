package kotsutil

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	troubleshootloader "github.com/replicatedhq/troubleshoot/pkg/loader"
)

func LoadTSKindsFromPath(dir string) (*troubleshootloader.TroubleshootKinds, error) {
	var renderedFiles []string
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrapf(err, "failed to walk dir: %s", dir)
			}

			if info.IsDir() {
				return nil
			}

			contents, err := ioutil.ReadFile(path)
			if err != nil {
				return errors.Wrap(err, "failed to read file")
			}

			renderedFiles = append(renderedFiles, string(contents))

			return nil
		})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to walk dir: %s", dir)
	}

	ctx := context.Background()
	tsKinds, err := troubleshootloader.LoadSpecs(ctx, troubleshootloader.LoadOptions{
		RawSpecs: renderedFiles,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to load troubleshoot specs from archive")
	}
	return tsKinds, nil
}
