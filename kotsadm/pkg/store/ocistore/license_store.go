package ocistore

import (
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

// GetInitialLicenseForApp reads from the app, not app version
func (s OCIStore) GetInitialLicenseForApp(appID string) (*kotsv1beta1.License, error) {
	app, err := s.GetApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(app.License), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse license")
	}
	license := obj.(*kotsv1beta1.License)

	return license, nil
}

func (s OCIStore) GetLatestLicenseForApp(appID string) (*kotsv1beta1.License, error) {
	return nil, ErrNotImplemented
}

func (s OCIStore) GetLicenseForAppVersion(appID string, sequence int64) (*kotsv1beta1.License, error) {
	appVersion, err := s.GetAppVersion(appID, sequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version")
	}

	if appVersion == nil {
		return nil, nil
	}

	return appVersion.KOTSKinds.License, nil
}
