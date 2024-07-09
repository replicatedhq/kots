package replicatedapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ApplicationMetadata struct {
	Manifest []byte
	Branding []byte
}

const DefaultMetadata = `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: "default-application"
spec:
  title: "the application"
  icon: https://cdn2.iconfinder.com/data/icons/mixd/512/16_kubernetes-512.png
  releaseNotes: |
    release notes`

type LicenseData struct {
	LicenseBytes []byte
	License      *kotsv1beta1.License
}

func GetPreferredChannelSlug() (string, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to get clientset")
	}

	namespace := util.AppNamespace()

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), types.KotsadmConfigMap, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get existing kotsadm config map")
	}

	slug, ok := configMap.Data["preferred-channel-slug"]
	if !ok {
		return "", nil
	}

	return slug, nil
}

func GetLatestLicense(license *kotsv1beta1.License) (*LicenseData, error) {
	installParams, err := kotsutil.GetInstallationParams()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installation params")
	}

	url := fmt.Sprintf("%s/license/%s", license.Spec.Endpoint, license.Spec.AppSlug)
	if installParams.PreferredChannelSlug != "" {
		url = fmt.Sprintf("%s/%s", url, installParams.PreferredChannelSlug)
	}

	licenseData, err := getLicenseFromAPI(url, license.Spec.LicenseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get license from api")
	}

	return licenseData, nil
}

func getAppIdFromLicenseId(s store.Store, licenseID string) (string, error) {
	apps, err := s.ListInstalledApps()
	if err != nil {
		return "", errors.Wrap(err, "failed to get all app licenses")
	}

	for _, a := range apps {
		l, err := s.GetLatestLicenseForApp(a.ID)
		if err != nil {
			return "", errors.Wrap(err, "failed to get latest license for app")
		}

		if l.Spec.LicenseID == licenseID {
			return a.ID, nil
		}
	}

	return "", nil
}

func getLicenseFromAPI(url string, licenseID string) (*LicenseData, error) {
	req, err := util.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	req.SetBasicAuth(licenseID, licenseID)

	if persistence.IsInitialized() {
		appId, err := getAppIdFromLicenseId(store.GetStore(), licenseID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get license by id")
		}

		if appId != "" {
			reportingInfo := reporting.GetReportingInfo(appId)
			reporting.InjectReportingInfoHeaders(req, reportingInfo)
		}
	}

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

// GetApplicationMetadata will return any available application yaml from
// the upstream. If there is no application.yaml, it will return
// a placeholder one
func GetApplicationMetadata(upstream *url.URL, versionLabel string) (*ApplicationMetadata, error) {
	host := util.GetReplicatedAPIEndpoint()
	manifest, err := getApplicationMetadataFromHost(host, "metadata", upstream, versionLabel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get metadata from %s", host)
	}

	if len(manifest) == 0 {
		manifest = []byte(DefaultMetadata)
	}

	branding, err := getApplicationMetadataFromHost(host, "branding", upstream, versionLabel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get branding from %s", host)
	}

	return &ApplicationMetadata{
		Manifest: manifest,
		Branding: branding,
	}, nil
}

func getApplicationMetadataFromHost(host string, endpoint string, upstream *url.URL, versionLabel string) ([]byte, error) {
	r, err := ParseReplicatedURL(upstream)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse replicated upstream")
	}

	if r.VersionLabel != nil && *r.VersionLabel != "" && versionLabel != "" && *r.VersionLabel != versionLabel {
		return nil, errors.Errorf("version label in upstream (%q) does not match version label in parameter (%q)", *r.VersionLabel, versionLabel)
	}

	getUrl := fmt.Sprintf("%s/%s/%s", host, endpoint, url.PathEscape(r.AppSlug))

	if r.Channel != nil {
		getUrl = fmt.Sprintf("%s/%s", getUrl, url.PathEscape(*r.Channel))
	}

	v := url.Values{}
	if r.VersionLabel != nil && *r.VersionLabel != "" {
		v.Set("versionLabel", *r.VersionLabel)
	} else if versionLabel != "" {
		v.Set("versionLabel", versionLabel)
	}
	getUrl = fmt.Sprintf("%s?%s", getUrl, v.Encode())

	getReq, err := util.NewRequest("GET", getUrl, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer getResp.Body.Close()

	if getResp.StatusCode == 404 {
		// no metadata is not an error
		return nil, nil
	}

	if getResp.StatusCode >= 400 {
		return nil, errors.Errorf("unexpected result from get request: %d", getResp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	return respBody, nil
}

func SendCustomAppMetricsData(license *kotsv1beta1.License, app *apptypes.App, data map[string]interface{}) error {
	url := fmt.Sprintf("%s/application/custom-metrics", license.Spec.Endpoint)

	payload := struct {
		Data map[string]interface{} `json:"data"`
	}{
		Data: data,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "marshal data")
	}

	req, err := util.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return errors.Wrap(err, "call newrequest")
	}
	req.Header.Set("Content-Type", "application/json")

	reportingInfo := reporting.GetReportingInfo(app.ID)
	reporting.InjectReportingInfoHeaders(req, reportingInfo)

	req.SetBasicAuth(license.Spec.LicenseID, license.Spec.LicenseID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Warnf("failed to read metrics response body: %v", err)
		}

		return errors.Errorf("unexpected result from post request: %d, data: %s", resp.StatusCode, respBody)
	}

	return nil
}
