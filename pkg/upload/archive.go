package upload

import (
	"io/ioutil"
	"path"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

func createUploadableArchive(rootPath string) (string, error) {
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}

	paths := []string{
		rootPath,
	}

	// the caller of this function is repsonsible for deleting this file
	tempDir, err := ioutil.TempDir("", "kots")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	if err := tarGz.Archive(paths, path.Join(tempDir, "kots-uploadable-archive.tar.gz")); err != nil {
		return "", errors.Wrap(err, "failed to create tar gz")
	}

	return path.Join(tempDir, "kots-uploadable-archive.tar.gz"), nil
}
