package replicatedapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	"github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
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
	License      *licensewrapper.LicenseWrapper
}

// GetLatestLicense will return the latest license from the replicated api, if selectedChannelID is provided
// it will be passed along to the api.
// Note: The Replicated API can return v1beta1 or v1beta2 licenses, which are wrapped in a LicenseWrapper.
func GetLatestLicense(license *licensewrapper.LicenseWrapper, selectedChannelID string) (*LicenseData, error) {
	if license == nil || (!license.IsV1() && !license.IsV2()) {
		return nil, errors.New("license wrapper contains no license")
	}

	fullURL, err := makeLicenseURL(license, selectedChannelID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make license url")
	}

	licenseData, err := getLicenseFromAPI(fullURL, license.GetLicenseID())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get license from api")
	}

	return licenseData, nil
}

func makeLicenseURL(license *licensewrapper.LicenseWrapper, selectedChannelID string) (string, error) {
	if license == nil || (!license.IsV1() && !license.IsV2()) {
		return "", errors.New("license wrapper contains no license")
	}

	// Use wrapper methods to access license fields
	endpoint := license.GetEndpoint()
	if endpoint == "" {
		endpoint = "https://replicated.app"
	}

	u, err := url.Parse(fmt.Sprintf("%s/license/%s", endpoint, license.GetAppSlug()))
	if err != nil {
		return "", errors.Wrap(err, "failed to parse url")
	}

	params := url.Values{}
	params.Add("licenseSequence", fmt.Sprintf("%d", license.GetLicenseSequence()))
	if selectedChannelID != "" {
		params.Add("selectedChannelId", selectedChannelID)
	}
	u.RawQuery = params.Encode()
	return u.String(), nil
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

		if l.GetLicenseID() == licenseID {
			return a.ID, nil
		}
	}

	return "", nil
}

func getLicenseFromAPI(url string, licenseID string) (*LicenseData, error) {
	req, err := util.NewRetryableRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	req.SetBasicAuth(licenseID, licenseID)

	if persistence.IsInitialized() && !util.IsUpgradeService() {
		appId, err := getAppIdFromLicenseId(store.GetStore(), licenseID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get license by id")
		}

		if appId != "" {
			reportingInfo := reporting.GetReportingInfo(appId)
			reporting.InjectReportingInfoHeaders(req.Header, reportingInfo)
		}
	}

	resp, err := util.DefaultHTTPClient.Do(req)
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
	obj, gvk, err := decode(body, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode latest license data")
	}

	// Wrap the license based on its version
	var wrapper *licensewrapper.LicenseWrapper
	if gvk.Version == "v1beta1" {
		wrapper = &licensewrapper.LicenseWrapper{
			V1: obj.(*kotsv1beta1.License),
		}
	} else if gvk.Version == "v1beta2" {
		wrapper = &licensewrapper.LicenseWrapper{
			V2: obj.(*kotsv1beta2.License),
		}
	} else {
		return nil, errors.Errorf("unsupported license version: %s", gvk.Version)
	}

	data := &LicenseData{
		LicenseBytes: body,
		License:      wrapper,
	}
	return data, nil
}

// GetApplicationMetadata will return any available application yaml from
// the upstream. If there is no application.yaml, it will return
// a placeholder one
func GetApplicationMetadata(host string, upstream *url.URL, versionLabel string) (*ApplicationMetadata, error) {
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

	getReq, err := util.NewRetryableRequest("GET", getUrl, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	getResp, err := util.DefaultHTTPClient.Do(getReq)
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

func SendCustomAppMetricsData(license *licensewrapper.LicenseWrapper, app *apptypes.App, data map[string]interface{}) error {
	endpoint := util.ReplicatedAppEndpoint(license)
	url := fmt.Sprintf("%s/application/custom-metrics", endpoint)

	payload := struct {
		Data map[string]interface{} `json:"data"`
	}{
		Data: data,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "marshal data")
	}

	req, err := util.NewRetryableRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return errors.Wrap(err, "call newrequest")
	}
	req.Header.Set("Content-Type", "application/json")

	reportingInfo := reporting.GetReportingInfo(app.ID)
	reporting.InjectReportingInfoHeaders(req.Header, reportingInfo)

	// Use wrapper method to get license ID
	licenseID := license.GetLicenseID()
	req.SetBasicAuth(licenseID, licenseID)

	resp, err := util.DefaultHTTPClient.Do(req)
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
