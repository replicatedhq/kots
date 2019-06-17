package templates

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func BuildDir(buildPath string, fs *afero.Afero, builder *Builder) error {
	isDir, err := fs.IsDir(buildPath)
	if err != nil {
		return errors.Wrapf(err, "check if dir %s", buildPath)
	}
	if !isDir {
		return buildFile(buildPath, fs, builder)
	}

	files, err := fs.ReadDir(buildPath)
	if err != nil {
		return errors.Wrapf(err, "read dir %s", buildPath)
	}
	for _, file := range files {
		childPath := filepath.Join(buildPath, file.Name())
		if file.IsDir() {
			err = BuildDir(childPath, fs, builder)
			if err != nil {
				return errors.Wrapf(err, "build dir %s", childPath)
			}
		} else {
			err = buildFile(childPath, fs, builder)
			if err != nil {
				return errors.Wrapf(err, "build file %s", childPath)
			}
		}
	}

	return nil
}

func buildFile(buildPath string, fs *afero.Afero, builder *Builder) error {
	fileContents, err := fs.ReadFile(buildPath)
	if err != nil {
		return errors.Wrapf(err, "read file %s", buildPath)
	}

	newContents, err := builder.String(string(fileContents))
	if err != nil {
		return errors.Wrapf(err, "template file %s", buildPath)
	}

	err = fs.WriteFile(buildPath, []byte(newContents), os.FileMode(777))
	if err != nil {
		return errors.Wrapf(err, "write file %s", buildPath)
	}
	return nil
}
