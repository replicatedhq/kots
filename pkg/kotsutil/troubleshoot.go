package kotsutil

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	troubleshootloader "github.com/replicatedhq/troubleshoot/pkg/loader"
)

func LoadTSKindsFromPath(dir string) (*troubleshootloader.TroubleshootKinds, error) {
	var renderedFiles []string
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrap(err, "failed to walk rendered dir")
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
		return nil, errors.Wrap(err, "failed to walk rendered dir")
	}

	ctx := context.Background()
	tsKinds, err := troubleshootloader.LoadSpecs(ctx, troubleshootloader.LoadOptions{
		RawSpec: strings.Join(renderedFiles, "\n---\n"),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to load troubleshoot specs from archive")
	}
	return tsKinds, nil
}
