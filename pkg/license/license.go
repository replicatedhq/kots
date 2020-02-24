package license

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func Sync(a *app.App, licenseData string) (*kotsv1beta1.License, error) {
	archiveDir, err := app.GetAppVersionArchive(a.ID, a.CurrentSequence)
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
		if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"), b.Bytes(), 0644); err != nil {
			return nil, errors.Wrap(err, "failed to write new license")
		}

		newSequence, err := a.CreateVersion(archiveDir, "License Change")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create new version")
		}

		if err := app.CreateAppVersionArchive(a.ID, newSequence, archiveDir); err != nil {
			return nil, errors.Wrap(err, "failed to upload")
		}
	}

	return latestLicense, nil
}
