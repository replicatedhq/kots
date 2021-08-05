package cluster

import "github.com/pkg/errors"

func installCRI() error {
	if err := verifyRuncInstallation(); err != nil {
		return errors.Wrap(err, "verify runc")
	}

	if err := verifyContainerdInstallation(); err != nil {
		return errors.Wrap(err, "verify containerd")
	}

	return nil
}
