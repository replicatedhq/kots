package license

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
)

type LicenseData struct {
	LicenseBytes []byte
	License      *kotsv1beta1.License
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
		if l.Spec.LicenseID == newLicense.Spec.LicenseID {
			return false, nil
		}
	}

	return true, nil
}

func GetLatestLicense(license *kotsv1beta1.License) (*LicenseData, error) {
	url := fmt.Sprintf("%s/license/%s", license.Spec.Endpoint, license.Spec.AppSlug)

	licenseData, err := getLicenseFromAPI(url, license.Spec.LicenseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get license from api")
	}

	return licenseData, nil
}

func GetLatestLicenseForHelm(licenseID string) (*LicenseData, error) {
	url := fmt.Sprintf("%s/license", util.GetReplicatedAPIEndpoint())
	licenseData, err := getLicenseFromAPI(url, licenseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get license from api")
	}

	return licenseData, nil
}

func getLicenseFromAPI(url string, licenseID string) (*LicenseData, error) {
	req, err := util.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	req.SetBasicAuth(licenseID, licenseID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load response")
	}

	if resp.StatusCode >= 400 {
		return nil, errors.Errorf("unexpected result from get request: %d, data: %s", resp.StatusCode, body)
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(body, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode latest license data")
	}

	data := &LicenseData{
		LicenseBytes: body,
		License:      obj.(*kotsv1beta1.License),
	}
	return data, nil
}
