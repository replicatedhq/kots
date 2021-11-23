package ocistore

import (
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	gitopstypes "github.com/replicatedhq/kots/pkg/gitops/types"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
)

func (s *OCIStore) GetLatestLicenseForApp(appID string) (*kotsv1beta1.License, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) GetLicenseForAppVersion(appID string, sequence int64) (*kotsv1beta1.License, error) {
	appVersion, err := s.GetAppVersion(appID, sequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version")
	}

	if appVersion == nil {
		return s.GetLatestLicenseForApp(appID)
	}

	return appVersion.KOTSKinds.License, nil
}

func (s *OCIStore) GetAllAppLicenses() ([]*kotsv1beta1.License, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) UpdateAppLicense(appID string, sequence int64, archiveDir string, newLicense *kotsv1beta1.License, originalLicenseData string, channelChanged bool, failOnVersionCreate bool, gitops gitopstypes.DownstreamGitOps, renderer rendertypes.Renderer) (int64, error) {
	return int64(0), ErrNotImplemented
}

func (s *OCIStore) UpdateAppLicenseSyncNow(appID string) error {
	return ErrNotImplemented
}
