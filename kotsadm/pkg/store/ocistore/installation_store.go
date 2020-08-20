package ocistore

import (
	installationtypes "github.com/replicatedhq/kots/kotsadm/pkg/online/types"
)

func (s OCIStore) GetPendingInstallationStatus() (*installationtypes.InstallStatus, error) {
	return nil, ErrNotImplemented
}
