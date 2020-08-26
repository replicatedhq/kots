package ocistore

import (
	airgaptypes "github.com/replicatedhq/kots/kotsadm/pkg/airgap/types"
)

func (s OCIStore) GetPendingAirgapUploadApp() (*airgaptypes.PendingApp, error) {
	return nil, ErrNotImplemented
}

func (s OCIStore) GetAirgapInstallStatus() (*airgaptypes.InstallStatus, error) {
	return nil, ErrNotImplemented
}

func (s OCIStore) ResetAirgapInstallInProgress(appID string) error {
	return ErrNotImplemented
}

func (s OCIStore) SetAppIsAirgap(appID string) error {
	return ErrNotImplemented
}

func (s OCIStore) SetAppInstallState(appID string, state string) error {
	return ErrNotImplemented
}
