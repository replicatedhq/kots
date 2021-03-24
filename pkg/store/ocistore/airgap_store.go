package ocistore

import (
	"github.com/pkg/errors"
	airgaptypes "github.com/replicatedhq/kots/pkg/airgap/types"
)

func (s *OCIStore) GetPendingAirgapUploadApp() (*airgaptypes.PendingApp, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) GetAirgapInstallStatus(appID string) (*airgaptypes.InstallStatus, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) ResetAirgapInstallInProgress(appID string) error {
	return ErrNotImplemented
}

func (s *OCIStore) SetAppIsAirgap(appID string, isAirgap bool) error {
	app, err := s.GetApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	app.IsAirgap = isAirgap

	if err := s.updateApp(app); err != nil {
		return errors.Wrap(err, "failed to update app")
	}

	return nil
}
