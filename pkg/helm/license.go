package helm

import (
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
)

func SyncLicense(helmApp *apptypes.HelmApp) (bool, error) {
	isSynced := false

	currentLicense, err := GetChartLicenseFromSecretOrDownload(helmApp)
	if err != nil {
		return isSynced, errors.Wrap(err, "failed to get license from secret")
	}

	licenseID := GetKotsLicenseID(&helmApp.Release)

	if currentLicense == nil && licenseID == "" {
		return isSynced, errors.Errorf("no license found for release %s", helmApp.Release.Name)
	}

	if licenseID == "" {
		licenseID = currentLicense.Spec.LicenseID
	} else if currentLicense != nil && licenseID != currentLicense.Spec.LicenseID {
		return isSynced, errors.Errorf("license ID in the chart does not match license ID in secret for release %s", helmApp.Release.Name)
	}

	latestLicenseData, err := replicatedapp.GetLatestLicenseForHelm(licenseID)
	if err != nil {
		return isSynced, errors.Wrap(err, "failed to get latest license for helm app")
	}

	err = SaveChartLicenseInSecret(helmApp, latestLicenseData.LicenseBytes)
	if err != nil {
		return isSynced, errors.Wrap(err, "failed to update helm license")
	}

	if currentLicense == nil {
		isSynced = true
	} else if currentLicense.Spec.LicenseSequence != latestLicenseData.License.Spec.LicenseSequence {
		isSynced = true
	}

	return isSynced, nil
}
