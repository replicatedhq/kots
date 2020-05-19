package license

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/registry"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func Sync(a *app.App, licenseData string) (*kotsv1beta1.License, error) {
	archiveDir, err := version.GetAppVersionArchive(a.ID, a.CurrentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest app version")
	}
	defer os.RemoveAll(archiveDir)

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kots kinds from archive")
	}

	if kotsKinds.License == nil {
		// this is not an error if there is no license
		return nil, nil
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

		appSequence, err := version.GetNextAppSequence(a.ID, &a.CurrentSequence)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get new app sequence")
		}

		registrySettings, err := registry.GetRegistrySettingsForApp(a.ID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get registry settings for app")
		}

		if err := render.RenderDir(archiveDir, a.ID, appSequence, registrySettings); err != nil {
			return nil, errors.Wrap(err, "failed to render new version")
		}

		newSequence, err := version.CreateVersion(a.ID, archiveDir, "License Change", a.CurrentSequence)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create new version")
		}

		if err := version.CreateAppVersionArchive(a.ID, newSequence, archiveDir); err != nil {
			return nil, errors.Wrap(err, "failed to upload")
		}

		if err := preflight.Run(a.ID, newSequence, archiveDir); err != nil {
			return nil, errors.Wrap(err, "failed to run preflights")
		}
	}

	return latestLicense, nil
}

func GetCurrentLicenseString(a *app.App) (string, error) {
	archiveDir, err := version.GetAppVersionArchive(a.ID, a.CurrentSequence)
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
