package ocistore

import (
	"github.com/pkg/errors"
	airgaptypes "github.com/replicatedhq/kots/kotsadm/pkg/airgap/types"
)

func (s OCIStore) GetPendingAirgapUploadApp() (*airgaptypes.PendingApp, error) {
	return nil, errors.New("not implemented")
}

func (s OCIStore) GetAirgapInstallStatus() (*airgaptypes.InstallStatus, error) {
	return nil, errors.New("not implemented")
}

func (s OCIStore) SetAppIsAirgap(appID string) error {
	return errors.New("not implemented")
}

func (s OCIStore) SetAppInstallState(appID string, state string) error {
	return errors.New("not implemented")
}
