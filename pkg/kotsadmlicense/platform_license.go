package license

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
)

type GetFromPlatformLicenseRequest struct {
	License string `json:"license"`
}

func GetFromPlatformLicense(apiEndpoint, platformLicense string) (string, error) {
	url := fmt.Sprintf("%s/license/platform", apiEndpoint)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(GetFromPlatformLicenseRequest{
		License: platformLicense,
	}); err != nil {
		return "", errors.Wrap(err, "failed to encode payload")
	}

	req, err := util.NewRequest("POST", url, &buf)
	if err != nil {
		return "", errors.Wrap(err, "failed to call newrequest")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute post request")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", errors.Wrap(err, "unexpected result from post request")
	}

	kotsLicenseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to load response")
	}

	return string(kotsLicenseData), nil
}
