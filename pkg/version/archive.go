package version

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/archiveutil"
)

func ExtractArchiveToTempDirectory(archiveFilename string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	if err := archiveutil.ExtractTGZ(context.TODO(), archiveFilename, tmpDir); err != nil {
		return "", errors.Wrap(err, "failed to extract archive")
	}

	return tmpDir, nil
}
