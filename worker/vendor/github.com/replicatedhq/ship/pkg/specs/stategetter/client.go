package stategetter

import (
	"context"
	"encoding/base64"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

type StateGetter struct {
	Logger   log.Logger
	Contents *state.UpstreamContents
	Fs       afero.Afero
}

func NewStateGetter(fs afero.Afero, logger log.Logger, contents *state.UpstreamContents) *StateGetter {
	return &StateGetter{
		Contents: contents,
		Fs:       fs,
		Logger:   logger,
	}
}

func (g *StateGetter) GetFiles(
	ctx context.Context,
	upstream string,
	destinationPath string,
) (string, error) {
	stateUnpackPath := filepath.Join(destinationPath, "state")

	for _, upstreamFile := range g.Contents.UpstreamFiles {
		err := g.Fs.MkdirAll(filepath.Join(stateUnpackPath, filepath.Dir(upstreamFile.FilePath)), 0755)
		if err != nil {
			return "", errors.Wrapf(err, "create dir for file %s", upstreamFile.FilePath)
		}

		rawContents, err := base64.StdEncoding.DecodeString(upstreamFile.FileContents)
		if err != nil {
			return "", errors.Wrapf(err, "decode contents of file %s", upstreamFile.FilePath)
		}

		err = g.Fs.WriteFile(filepath.Join(stateUnpackPath, upstreamFile.FilePath), rawContents, 0755)
		if err != nil {
			return "", errors.Wrapf(err, "write contents of file %s", upstreamFile.FilePath)
		}
	}

	return stateUnpackPath, nil
}
