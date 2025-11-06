package license

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
)

func Sync(a *apptypes.App, licenseString string, failOnVersionCreate bool) (*licensewrapper.LicenseWrapper, bool, error) {
	latestSequence, err := store.GetStore().GetLatestAppSequence(a.ID, true)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get latest app sequence")
	}

	currentLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get current license")
	}

	var updatedLicense *licensewrapper.LicenseWrapper
	if licenseString != "" {
		// Load and verify license using wrapper (supports both v1beta1 and v1beta2)
		unverifiedLicense, err := licensewrapper.LoadLicenseFromBytes([]byte(licenseString))
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to parse license")
		}

		verifiedLicense, err := kotslicense.VerifyLicenseWrapper(&unverifiedLicense)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to verify license")
		}

		updatedLicense = verifiedLicense
	} else {
		// get from the api
		licenseData, err := replicatedapp.GetLatestLicense(currentLicense, a.SelectedChannelID)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to get latest license")
		}
		updatedLicense = licenseData.License
		licenseString = string(licenseData.LicenseBytes)
	}

	// check to see if both licenses are of the 'serviceaccount token' type, and if so check if the account ID matches
	_, serviceAccountUpdated, saMatchErr := ValidateServiceAccountToken(updatedLicense.GetLicenseID(), currentLicense)

	if currentLicense.GetLicenseID() != updatedLicense.GetLicenseID() && saMatchErr != nil {
		return nil, false, errors.New("license ids do not match")
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	// Because an older version can be edited, it is possible to have latest version with an outdated license.
	// So even if global license sequence is already latest, we still need to create a new app version in this case.
	err = store.GetStore().GetAppVersionArchive(a.ID, latestSequence, archiveDir)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get latest app sequence")
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to load kotskinds from path")
	}

	synced := false
	if updatedLicense.GetLicenseSequence() != currentLicense.GetLicenseSequence() ||
		updatedLicense.GetLicenseSequence() != kotsKinds.License.GetLicenseSequence() ||
		serviceAccountUpdated {

		channelChanged := false
		if updatedLicense.GetChannelID() != currentLicense.GetChannelID() {
			channelChanged = true
		}
		reportingInfo := reporting.GetReportingInfo(a.ID)
		newSequence, err := store.GetStore().UpdateAppLicense(a.ID, latestSequence, archiveDir, updatedLicense, licenseString, channelChanged, failOnVersionCreate, &render.Renderer{}, reportingInfo)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to update license")
		}

		if err := preflight.Run(a.ID, a.Slug, newSequence, a.IsAirgap, false, archiveDir); err != nil {
			return nil, false, errors.Wrap(err, "failed to run preflights")
		}
		synced = true
	} else {
		err := store.GetStore().UpdateAppLicenseSyncNow(a.ID)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to update license sync time")
		}
	}

	return updatedLicense, synced, nil
}

func SyncWithServiceAccountToken(a *apptypes.App, serviceAccountToken string, failOnVersionCreate bool) (*licensewrapper.LicenseWrapper, bool, error) {
	currentLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get current license")
	}

	licenseWithToken := currentLicense.V1.DeepCopy()
	licenseWithToken.Spec.LicenseID = serviceAccountToken
	wrappedLicenseWithToken := &licensewrapper.LicenseWrapper{V1: licenseWithToken}

	licenseData, err := replicatedapp.GetLatestLicense(wrappedLicenseWithToken, a.SelectedChannelID)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get latest license with service account token")
	}

	return Sync(a, string(licenseData.LicenseBytes), failOnVersionCreate)
}

func Change(a *apptypes.App, newLicenseString string) (*licensewrapper.LicenseWrapper, error) {
	if newLicenseString == "" {
		return nil, errors.New("license cannot be empty")
	}

	// Load and verify license using wrapper (supports both v1beta1 and v1beta2)
	unverifiedLicense, err := licensewrapper.LoadLicenseFromBytes([]byte(newLicenseString))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load license from bytes")
	}

	newLicense, err := kotslicense.VerifyLicenseWrapper(&unverifiedLicense)
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify license")
	}

	if !a.IsAirgap {
		licenseData, err := replicatedapp.GetLatestLicense(newLicense, a.SelectedChannelID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get latest license")
		}
		newLicense = licenseData.License
		newLicenseString = string(licenseData.LicenseBytes)
	} else {
		// check if new license supports airgap mode
		if !newLicense.IsAirgapSupported() {
			return nil, errors.New("New license does not support airgapped installations")
		}
	}

	// Check if license is expired (works for both v1beta1 and v1beta2)
	expired, err := kotslicense.LicenseIsExpired(newLicense)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if license is expired")
	}
	if expired {
		return nil, errors.New("License is expired")
	}

	currentLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current license")
	}

	if currentLicense.GetLicenseType() != "community" {
		return nil, errors.New("Changing from a non-community license is not supported")
	}
	if currentLicense.GetLicenseID() == newLicense.GetLicenseID() {
		return nil, errors.New("New license is the same as the current license")
	}
	if currentLicense.GetAppSlug() != newLicense.GetAppSlug() {
		return nil, errors.New("New license is for a different application")
	}

	// check if license already exists
	existingLicense, err := CheckIfLicenseExists([]byte(newLicenseString))
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if license exists")
	}
	if existingLicense != nil {
		resolved, err := ResolveExistingLicense(newLicense)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to resolve existing license conflict"))
		}
		if !resolved {
			return nil, errors.New("License already exists")
		}
	}

	latestSequence, err := store.GetStore().GetLatestAppSequence(a.ID, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest app sequence")
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(a.ID, latestSequence, archiveDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest app sequence")
	}

	channelChanged := false
	if newLicense.GetChannelID() != currentLicense.GetChannelID() {
		channelChanged = true
	}
	reportingInfo := reporting.GetReportingInfo(a.ID)
	newSequence, err := store.GetStore().UpdateAppLicense(a.ID, latestSequence, archiveDir, newLicense, newLicenseString, channelChanged, true, &render.Renderer{}, reportingInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update license")
	}

	if err := preflight.Run(a.ID, a.Slug, newSequence, a.IsAirgap, false, archiveDir); err != nil {
		return nil, errors.Wrap(err, "failed to run preflights")
	}

	return newLicense, nil
}

func CheckIfLicenseExists(license []byte) (*licensewrapper.LicenseWrapper, error) {
	decodedLicense, err := licensewrapper.LoadLicenseFromBytes(license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode license yaml")
	}

	allLicenses, err := store.GetStore().GetAllAppLicenses()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get all app licenses")
	}

	for _, l := range allLicenses {
		if l.GetLicenseID() == decodedLicense.GetLicenseID() {
			return l, nil
		}
	}

	return nil, nil
}

func ResolveExistingLicense(newLicense *licensewrapper.LicenseWrapper) (bool, error) {
	if newLicense == nil || (!newLicense.IsV1() && !newLicense.IsV2()) {
		return false, errors.New("invalid license: must be v1beta1 or v1beta2")
	}

	notInstalledApps, err := store.GetStore().ListFailedApps()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list failed apps"))
		return false, err
	}

	for _, app := range notInstalledApps {
		appLicense, err := licensewrapper.LoadLicenseFromBytes([]byte(app.License))
		if err != nil {
			continue
		}

		if appLicense.GetLicenseID() != newLicense.GetLicenseID() {
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
		if l.GetLicenseID() == newLicense.GetLicenseID() {
			return false, nil
		}
	}

	return true, nil
}
