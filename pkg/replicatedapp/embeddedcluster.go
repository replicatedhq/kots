package replicatedapp

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func GetECVersionForRelease(license *kotsv1beta1.License, versionLabel string) (string, error) {
	url := fmt.Sprintf("%s/clusterconfig/version/Installer?versionLabel=%s", util.ReplicatedAppEndpoint(license), url.QueryEscape(versionLabel))
	req, err := util.NewRetryableRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to call newrequest")
	}

	req.SetBasicAuth(license.Spec.LicenseID, license.Spec.LicenseID)

	resp, err := util.DefaultHTTPClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.Errorf("unexpected status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read body")
	}

	response := struct {
		Version string `json:"version"`
	}{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal response")
	}

	return response.Version, nil
}

func DownloadKOTSBinary(license *kotsv1beta1.License, versionLabel string) (string, error) {
	url := fmt.Sprintf("%s/clusterconfig/artifact/kots?versionLabel=%s", util.ReplicatedAppEndpoint(license), url.QueryEscape(versionLabel))
	req, err := util.NewRetryableRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to call newrequest")
	}
	logger.Infof("downloading kots binary from %s", url)

	req.SetBasicAuth(license.Spec.LicenseID, license.Spec.LicenseID)

	client := retryablehttp.NewClient()
	client.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		req.Header.Del("Authorization")
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.Errorf("unexpected status code %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "kotsbin")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}
	defer tmpFile.Close()

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to get new gzip reader")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", errors.Wrap(err, "failed to get read archive")
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}
		if header.Name != "kots" {
			continue
		}

		if _, err := io.Copy(tmpFile, tarReader); err != nil {
			return "", errors.Wrap(err, "failed to copy kots binary")
		}
		if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
			return "", errors.Wrap(err, "failed to set file permissions")
		}

		return tmpFile.Name(), nil
	}

	return "", errors.New("kots binary not found in archive")
}
