package version

import (
	"os"

	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
)

func ExtractArchiveToTempDirectory(archiveFilename string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "kotsadm")
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
