package upload

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

func createUploadableArchive(rootPath string) (string, error) {
	if strings.HasSuffix(rootPath, string(os.PathSeparator)) {
		rootPath = strings.TrimSuffix(rootPath, string(os.PathSeparator))
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}

	paths := []string{
		path.Join(rootPath, "upstream"),
		path.Join(rootPath, "base"),
		path.Join(rootPath, "overlays"),
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
