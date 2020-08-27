package ocistore

import (
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

func (s OCIStore) GetInitialLicenseForApp(appID string) (*kotsv1beta1.License, error) {
	return nil, ErrNotImplemented
}

func (s OCIStore) GetLatestLicenseForApp(appID string) (*kotsv1beta1.License, error) {
	return nil, ErrNotImplemented
}
