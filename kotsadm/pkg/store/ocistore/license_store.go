package ocistore

import (
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

func (s OCIStore) GetLatestLicenseForApp(appID string) (*kotsv1beta1.License, error) {
	return nil, ErrNotImplemented
}

func (s OCIStore) GetLicenseForAppVersion(appID string, sequence int64) (*kotsv1beta1.License, error) {
	appVersion, err := s.GetAppVersion(appID, sequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version")
	}

	if appVersion == nil {
		return s.GetLatestLicenseForApp(appID)
	}

	return appVersion.KOTSKinds.License, nil
}

func (s OCIStore) GetAllAppLicenses() ([]*kotsv1beta1.License, error) {
	return nil, ErrNotImplemented
}
