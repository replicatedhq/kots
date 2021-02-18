package version

import (
	"io/ioutil"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

func ExtractArchiveToTempDirectory(archiveFilename string) (string, error) {
	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
			OverwriteExisting:      true,
		},
	}
	if err := tarGz.Unarchive(archiveFilename, tmpDir); err != nil {
		return "", errors.Wrap(err, "failed to unarchive")
	}

	return tmpDir, nil
}
