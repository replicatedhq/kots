package license

import (
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
)

func ResolveExistingLicense(newLicense *kotsv1beta1.License) (bool, error) {
	notInstalledApps, err := store.GetStore().ListFailedApps()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list failed apps"))
		return false, err
	}

	for _, app := range notInstalledApps {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(app.License), nil, nil)
		if err != nil {
			continue
		}
		license := obj.(*kotsv1beta1.License)
		if license.Spec.LicenseID != newLicense.Spec.LicenseID {
			continue
		}

		if err := store.GetStore().RemoveApp(app.ID); err != nil {
			return false, errors.Wrap(err, "failed to remove existing app record")
		}
	}

	// check if license still exists
	allLicenses, err := store.GetStore().GetAllAppLicenses()
	if err != nil {
		return false, errors.Wrap(err, "failed to get all app licenses")
	}
	for _, l := range allLicenses {
		if l.Spec.LicenseID == newLicense.Spec.LicenseID {
			return false, nil
		}
	}

	return true, nil
}

func LicenseIsExpired(license *kotsv1beta1.License) (bool, error) {
	val, found := license.Spec.Entitlements["expires_at"]
	if !found {
		return false, nil
	}
	if val.ValueType != "" && val.ValueType != "String" {
		return false, errors.Errorf("expires_at must be type String: %s", val.ValueType)
	}
	if val.Value.StrVal == "" {
		return false, nil
	}

	partsed, err := time.Parse(time.RFC3339, val.Value.StrVal)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse expiration time")
	}
	return partsed.Before(time.Now()), nil
}
