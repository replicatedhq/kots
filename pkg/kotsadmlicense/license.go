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
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"k8s.io/client-go/kubernetes/scheme"
)

func Sync(a *apptypes.App, licenseString string, failOnVersionCreate bool) (licensewrapper.LicenseWrapper, bool, error) {
	latestSequence, err := store.GetStore().GetLatestAppSequence(a.ID, true)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to get latest app sequence")
	}

	currentLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to get current license")
	}

	var updatedLicense licensewrapper.LicenseWrapper
	if licenseString != "" {
		// Load and verify license using wrapper (supports both v1beta1 and v1beta2)
		unverifiedLicense, err := licensewrapper.LoadLicenseFromBytes([]byte(licenseString))
		if err != nil {
			return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to parse license")
		}

		verifiedLicense, err := kotslicense.VerifyLicenseWrapper(unverifiedLicense)
		if err != nil {
			return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to verify license")
		}

		updatedLicense = verifiedLicense
	} else {
		// get from the api
		// Note: GetLatestLicense still expects v1beta1.License (will be updated in Phase 4)
		licenseData, err := replicatedapp.GetLatestLicense(currentLicense.V1, a.SelectedChannelID)
		if err != nil {
			return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to get latest license")
		}
		updatedLicense = licensewrapper.LicenseWrapper{V1: licenseData.License}
		licenseString = string(licenseData.LicenseBytes)
	}

	// check to see if both licenses are of the 'serviceaccount token' type, and if so check if the account ID matches
	_, serviceAccountUpdated, saMatchErr := ValidateServiceAccountToken(updatedLicense.GetLicenseID(), currentLicense.GetLicenseID())

	if currentLicense.GetLicenseID() != updatedLicense.GetLicenseID() && saMatchErr != nil {
		return licensewrapper.LicenseWrapper{}, false, errors.New("license ids do not match")
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	// Because an older version can be edited, it is possible to have latest version with an outdated license.
	// So even if global license sequence is already latest, we still need to create a new app version in this case.
	err = store.GetStore().GetAppVersionArchive(a.ID, latestSequence, archiveDir)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to get latest app sequence")
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to load kotskinds from path")
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
			return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to update license")
		}

		if err := preflight.Run(a.ID, a.Slug, newSequence, a.IsAirgap, false, archiveDir); err != nil {
			return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to run preflights")
		}
		synced = true
	} else {
		err := store.GetStore().UpdateAppLicenseSyncNow(a.ID)
		if err != nil {
			return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to update license sync time")
		}
	}

	return updatedLicense, synced, nil
}

func SyncWithServiceAccountToken(a *apptypes.App, serviceAccountToken string, failOnVersionCreate bool) (licensewrapper.LicenseWrapper, bool, error) {
	currentLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to get current license")
	}

	licenseWithToken := currentLicense.V1.DeepCopy()
	licenseWithToken.Spec.LicenseID = serviceAccountToken

	licenseData, err := replicatedapp.GetLatestLicense(licenseWithToken, a.SelectedChannelID)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, false, errors.Wrap(err, "failed to get latest license with service account token")
	}

	return Sync(a, string(licenseData.LicenseBytes), failOnVersionCreate)
}

func Change(a *apptypes.App, newLicenseString string) (licensewrapper.LicenseWrapper, error) {
	if newLicenseString == "" {
		return licensewrapper.LicenseWrapper{}, errors.New("license cannot be empty")
	}

	// Load and verify license using wrapper (supports both v1beta1 and v1beta2)
	unverifiedLicense, err := licensewrapper.LoadLicenseFromBytes([]byte(newLicenseString))
	if err != nil {
		return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to load license from bytes")
	}

	newLicense, err := kotslicense.VerifyLicenseWrapper(unverifiedLicense)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to verify license")
	}

	if !a.IsAirgap {
		// Note: GetLatestLicense still expects v1beta1.License (will be updated in Phase 4)
		licenseData, err := replicatedapp.GetLatestLicense(newLicense.V1, a.SelectedChannelID)
		if err != nil {
			return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to get latest license")
		}
		newLicense = licensewrapper.LicenseWrapper{V1: licenseData.License}
		newLicenseString = string(licenseData.LicenseBytes)
	} else {
		// check if new license supports airgap mode
		if !newLicense.IsAirgapSupported() {
			return licensewrapper.LicenseWrapper{}, errors.New("New license does not support airgapped installations")
		}
	}

	// Note: LicenseIsExpired still expects v1beta1.License (will be updated in Phase 4)
	var licenseForExpiryCheck *kotsv1beta1.License
	if newLicense.IsV1() {
		licenseForExpiryCheck = newLicense.V1
	} else {
		// Temporary conversion for expiry check
		licenseForExpiryCheck = newLicense.V1
	}
	expired, err := kotslicense.LicenseIsExpired(licenseForExpiryCheck)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to check if license is expired")
	}
	if expired {
		return licensewrapper.LicenseWrapper{}, errors.New("License is expired")
	}

	currentLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to get current license")
	}

	if currentLicense.GetLicenseType() != "community" {
		return licensewrapper.LicenseWrapper{}, errors.New("Changing from a non-community license is not supported")
	}
	if currentLicense.GetLicenseID() == newLicense.GetLicenseID() {
		return licensewrapper.LicenseWrapper{}, errors.New("New license is the same as the current license")
	}
	if currentLicense.GetAppSlug() != newLicense.GetAppSlug() {
		return licensewrapper.LicenseWrapper{}, errors.New("New license is for a different application")
	}

	// check if license already exists
	existingLicense, err := CheckIfLicenseExists([]byte(newLicenseString))
	if err != nil {
		return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to check if license exists")
	}
	if existingLicense != nil {
		// Note: ResolveExistingLicense expects v1beta1.License (temporary)
		var licenseForResolve *kotsv1beta1.License
		if newLicense.IsV1() {
			licenseForResolve = newLicense.V1
		} else {
			licenseForResolve = newLicense.V1
		}
		resolved, err := ResolveExistingLicense(licenseForResolve)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to resolve existing license conflict"))
		}
		if !resolved {
			return licensewrapper.LicenseWrapper{}, errors.New("License already exists")
		}
	}

	latestSequence, err := store.GetStore().GetLatestAppSequence(a.ID, true)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to get latest app sequence")
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(a.ID, latestSequence, archiveDir)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to get latest app sequence")
	}

	channelChanged := false
	if newLicense.GetChannelID() != currentLicense.GetChannelID() {
		channelChanged = true
	}
	reportingInfo := reporting.GetReportingInfo(a.ID)
	newSequence, err := store.GetStore().UpdateAppLicense(a.ID, latestSequence, archiveDir, newLicense, newLicenseString, channelChanged, true, &render.Renderer{}, reportingInfo)
	if err != nil {
		return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to update license")
	}

	if err := preflight.Run(a.ID, a.Slug, newSequence, a.IsAirgap, false, archiveDir); err != nil {
		return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to run preflights")
	}

	return newLicense, nil
}

func CheckIfLicenseExists(license []byte) (*kotsv1beta1.License, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(license, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode license yaml")
	}
	decodedLicense := obj.(*kotsv1beta1.License)

	allLicenses, err := store.GetStore().GetAllAppLicenses()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get all app licenses")
	}

	for _, l := range allLicenses {
		if l.GetLicenseID() == decodedLicense.Spec.LicenseID {
			return l.V1, nil
		}
	}

	return nil, nil
}

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
		if l.GetLicenseID() == newLicense.Spec.LicenseID {
			return false, nil
		}
	}

	return true, nil
}
