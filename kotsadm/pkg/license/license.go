package license

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	registrytypes "github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/reporting"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func Sync(a *apptypes.App, licenseString string, failOnVersionCreate bool) (*kotsv1beta1.License, error) {
	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(a.ID, a.CurrentSequence, archiveDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest app version")
	}

	currentLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current license")
	}

	var updatedLicense *kotsv1beta1.License
	if licenseString != "" {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(licenseString), nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse license")
		}

		unverifiedLicense := obj.(*kotsv1beta1.License)
		verifiedLicense, err := kotspull.VerifySignature(unverifiedLicense)
		if err != nil {
			return nil, errors.Wrap(err, "failed to verify license")
		}

		updatedLicense = verifiedLicense
	} else {
		// get from the api
		licenseData, err := kotslicense.GetLatestLicense(currentLicense)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get latest license")
		}
		updatedLicense = licenseData.License
		licenseString = string(licenseData.LicenseBytes)
	}

	// Save and make a new version if the sequence has changed
	if updatedLicense.Spec.LicenseSequence != currentLicense.Spec.LicenseSequence {
		s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
		var b bytes.Buffer
		if err := s.Encode(updatedLicense, &b); err != nil {
			return nil, errors.Wrap(err, "failed to encode license")
		}
		encodedLicense := b.Bytes()
		if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"), encodedLicense, 0644); err != nil {
			return nil, errors.Wrap(err, "failed to write new license")
		}

		//  app has the original license data received from the server
		if err := updateAppLicense(a, licenseString); err != nil {
			return nil, errors.Wrap(err, "update app license")
		}

		registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get registry settings for app")
		}

		if err := createNewVersion(a, archiveDir, registrySettings); err != nil {
			// ignore error here to prevent a failure to render the current version
			// preventing the end-user from updating the application
			if failOnVersionCreate {
				return nil, err
			}
			logger.Errorf("Failed to create new version from license sync: %v", err)
		}
	}

	return updatedLicense, nil
}

func createNewVersion(a *apptypes.App, archiveDir string, registrySettings *registrytypes.RegistrySettings) error {
	app, err := store.GetStore().GetApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}
	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to list downstreams")
	}

	if err := render.RenderDir(archiveDir, app, downstreams, registrySettings, reporting.GetReportingInfo(a.ID)); err != nil {
		return errors.Wrap(err, "failed to render new version")
	}

	newSequence, err := version.CreateVersion(a.ID, archiveDir, "License Change", a.CurrentSequence, false)
	if err != nil {
		return errors.Wrap(err, "failed to create new version")
	}

	if err := preflight.Run(a.ID, newSequence, a.IsAirgap, archiveDir); err != nil {
		return errors.Wrap(err, "failed to run preflights")
	}

	return nil
}

// Gets the license as it was at a given app sequence
func GetCurrentLicenseString(a *apptypes.App) (string, error) {
	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(a.ID, a.CurrentSequence, archiveDir)
	if err != nil {
		return "", errors.Wrap(err, "failed to get latest app version")
	}

	kotsLicense, err := ioutil.ReadFile(filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"))
	if err != nil {
		return "", errors.Wrap(err, "failed to read license file from archive")
	}
	return string(kotsLicense), nil
}

func CheckDoesLicenseExists(allLicenses []*kotsv1beta1.License, uploadedLicense string) (*kotsv1beta1.License, error) {
	parsedUploadedLicense, err := GetParsedLicense(uploadedLicense)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse uploaded license")
	}

	for _, license := range allLicenses {
		if license.Spec.LicenseID == parsedUploadedLicense.Spec.LicenseID {
			return license, nil
		}
	}
	return nil, nil
}

func GetParsedLicense(licenseStr string) (*kotsv1beta1.License, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(licenseStr), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode license yaml")
	}
	license := obj.(*kotsv1beta1.License)
	return license, nil
}
