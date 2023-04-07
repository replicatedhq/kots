package upstream

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/upstream/types"
)

func readFilesFromPath(filepath string) (*types.Upstream, error) {
	return nil, errors.New("readFilesFromPath not implemented")
}

func readFilesFromURI(upstreamURI string) (*types.Upstream, error) {
	return nil, errors.New("readFilesFromURI not implemented")
}

func ReadUpstreamFilesFromPath(dir string) ([]types.UpstreamFile, error) {
	upstreamFiles := []types.UpstreamFile{}
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return errors.Wrap(err, "failed to read file")
			}

			upstreamFile := types.UpstreamFile{
				Path:    path,
				Content: content,
			}

			upstreamFiles = append(upstreamFiles, upstreamFile)

			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk dir")
	}

	return upstreamFiles, nil
}
