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
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func Sync(a *apptypes.App, licenseData string, failOnVersionCreate bool) (*kotsv1beta1.License, error) {
	archiveDir, err := store.GetStore().GetAppVersionArchive(a.ID, a.CurrentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest app version")
	}
	defer os.RemoveAll(archiveDir)

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kots kinds from archive")
	}

	if kotsKinds.License == nil {
		return nil, errors.New("app does not contain a license")
	}

	latestLicense := kotsKinds.License
	if licenseData != "" {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(licenseData), nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse license")
		}

		unverifiedLicense := obj.(*kotsv1beta1.License)
		verifiedLicense, err := kotspull.VerifySignature(unverifiedLicense)
		if err != nil {
			return nil, errors.Wrap(err, "failed to verify license")
		}

		latestLicense = verifiedLicense
	} else {
		// get from the api
		updatedLicense, err := kotslicense.GetLatestLicense(kotsKinds.License)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get latest license")
		}
		latestLicense = updatedLicense
	}

	// Save and make a new version if the sequence has changed
	if latestLicense.Spec.LicenseSequence != kotsKinds.License.Spec.LicenseSequence {
		s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
		var b bytes.Buffer
		if err := s.Encode(latestLicense, &b); err != nil {
			return nil, errors.Wrap(err, "failed to encode license")
		}
		encodedLicense := b.Bytes()
		if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"), encodedLicense, 0644); err != nil {
			return nil, errors.Wrap(err, "failed to write new license")
		}

		if err := updateAppLicense(a, string(encodedLicense)); err != nil {
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

	return latestLicense, nil
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

	if err := render.RenderDir(archiveDir, app, downstreams, registrySettings); err != nil {
		return errors.Wrap(err, "failed to render new version")
	}

	newSequence, err := version.CreateVersion(a.ID, archiveDir, "License Change", a.CurrentSequence)
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
	archiveDir, err := store.GetStore().GetAppVersionArchive(a.ID, a.CurrentSequence)
	if err != nil {
		return "", errors.Wrap(err, "failed to get latest app version")
	}
	defer os.RemoveAll(archiveDir)

	kotsLicense, err := ioutil.ReadFile(filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"))
	if err != nil {
		return "", errors.Wrap(err, "failed to read license file from archive")
	}
	return string(kotsLicense), nil
}
