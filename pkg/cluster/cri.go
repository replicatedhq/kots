package cluster

import (
	"context"

	"github.com/pkg/errors"
)

func startCRI(dataDir string) error {
	if err := verifyRuncInstallation(); err != nil {
		return errors.Wrap(err, "verify runc")
	}

	if err := startContainerd(context.Background(), dataDir); err != nil {
		return errors.Wrap(err, "verify containerd")
	}

	return nil
}
