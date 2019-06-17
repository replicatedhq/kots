package localgetter

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type LocalGetter struct {
	Logger log.Logger
	FS     afero.Afero
}

func (g *LocalGetter) GetFiles(ctx context.Context, upstream, savePath string) (string, error) {
	debug := level.Debug(g.Logger)
	debug.Log("event", "localgetter.GetFiles", "upstream", upstream, "savePath", savePath)

	err := g.copyDir(ctx, upstream, savePath)
	if err != nil {
		return "", errors.Wrap(err, "copy files")
	}
	return savePath, nil
}

func (g *LocalGetter) copyDir(ctx context.Context, upstream, savePath string) error {
	isDir, err := g.FS.IsDir(upstream)
	if err != nil {
		return errors.Wrapf(err, "check if %s is dir", upstream)
	}
	if !isDir {
		// copy a single file
		return g.copyFile(ctx, upstream, savePath, os.FileMode(777))
	}

	files, err := g.FS.ReadDir(upstream)
	if err != nil {
		return errors.Wrapf(err, "read files in dir %s", upstream)
	}

	for _, file := range files {
		loopFile := filepath.Join(upstream, file.Name())
		loopDest := filepath.Join(savePath, file.Name())
		if file.IsDir() {
			err = g.FS.MkdirAll(loopDest, file.Mode())
			if err != nil {
				return errors.Wrapf(err, "create dest dir %s", loopDest)
			}

			err = g.copyDir(ctx, loopFile, loopDest)
			if err != nil {
				return errors.Wrapf(err, "copy dir %s", file.Name())
			}
		} else {
			err = g.copyFile(ctx, loopFile, loopDest, file.Mode())
			if err != nil {
				return errors.Wrapf(err, "copy file %s", file.Name())
			}
		}
	}
	return nil
}

func (g *LocalGetter) copyFile(ctx context.Context, upstream, savePath string, mode os.FileMode) error {
	saveDir := filepath.Dir(savePath)
	exists, err := g.FS.Exists(saveDir)
	if err != nil {
		return errors.Wrapf(err, "determine if path %s exists", saveDir)
	}
	if !exists {
		err = g.FS.MkdirAll(saveDir, os.ModePerm)
		if err != nil {
			return errors.Wrapf(err, "create dest dir %s", saveDir)
		}
	}

	contents, err := g.FS.ReadFile(upstream)
	if err != nil {
		return errors.Wrapf(err, "read %s file contents", upstream)
	}

	err = g.FS.WriteFile(savePath, contents, mode)
	return errors.Wrapf(err, "write %s file contents", savePath)
}

func IsLocalFile(FS *afero.Afero, upstream string) bool {
	exists, err := FS.Exists(upstream)
	if err != nil {
		return false
	}
	return exists
}
