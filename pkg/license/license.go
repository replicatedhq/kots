package license

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/version"
)

func GetLatestLicense(license *kotsv1beta1.License) (*kotsv1beta1.License, error) {
	url := fmt.Sprintf("%s/license/%s", license.Spec.Endpoint, license.Spec.AppSlug)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}
	req.Header.Add("User-Agent", fmt.Sprintf("KOTS/%s", version.Version()))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID)))))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, errors.Wrap(err, "unexpected result from get request")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load response")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(body, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode latest license data")
	}
	latestLicense := obj.(*kotsv1beta1.License)

	return latestLicense, nil
}
