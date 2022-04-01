package license

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/preflight"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/version"
	"k8s.io/client-go/kubernetes/scheme"
)

func Sync(a *apptypes.App, licenseString string, failOnVersionCreate bool) (*kotsv1beta1.License, bool, error) {
	latestSequence, err := store.GetStore().GetLatestAppSequence(a.ID, true)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get latest app sequence")
	}

	currentLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get current license")
	}

	var updatedLicense *kotsv1beta1.License
	if licenseString != "" {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(licenseString), nil, nil)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to parse license")
		}

		unverifiedLicense := obj.(*kotsv1beta1.License)
		verifiedLicense, err := kotspull.VerifySignature(unverifiedLicense)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to verify license")
		}

		updatedLicense = verifiedLicense
	} else {
		// get from the api
		licenseData, err := kotslicense.GetLatestLicense(currentLicense)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to get latest license")
		}
		updatedLicense = licenseData.License
		licenseString = string(licenseData.LicenseBytes)
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

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to load kotskinds from path")
	}

	synced := false
	if updatedLicense.Spec.LicenseSequence != currentLicense.Spec.LicenseSequence ||
		updatedLicense.Spec.LicenseSequence != kotsKinds.License.Spec.LicenseSequence {

		channelChanged := false
		if updatedLicense.Spec.ChannelID != currentLicense.Spec.ChannelID {
			channelChanged = true
		}
		newSequence, err := store.GetStore().UpdateAppLicense(a.ID, latestSequence, archiveDir, updatedLicense, licenseString, channelChanged, failOnVersionCreate, &version.DownstreamGitOps{}, &render.Renderer{})
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to update license")
		}

		if err := preflight.Run(a.ID, a.Slug, newSequence, a.IsAirgap, archiveDir); err != nil {
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
		if l.Spec.LicenseID == decodedLicense.Spec.LicenseID {
			return l, nil
		}
	}

	return nil, nil
}
