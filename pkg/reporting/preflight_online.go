package reporting

import (
	"bytes"
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
)

func (r *OnlineReporter) SubmitPreflightData(license *licensewrapper.LicenseWrapper, appID string, clusterID string, sequence int64, skipPreflights bool, installStatus storetypes.DownstreamVersionStatus, isCLI bool, preflightStatus string, appStatus string) error {
	endpoint := util.ReplicatedAppEndpoint(license)
	if !canReport(endpoint) {
		return nil
	}

	urlValues := url.Values{}
	urlValues.Set("sequence", fmt.Sprintf("%d", sequence))
	urlValues.Set("skipPreflights", fmt.Sprintf("%t", skipPreflights))
	urlValues.Set("installStatus", string(installStatus))
	urlValues.Set("isCLI", fmt.Sprintf("%t", isCLI))
	urlValues.Set("preflightStatus", preflightStatus)
	urlValues.Set("appStatus", appStatus)
	urlValues.Set("kotsVersion", buildversion.Version())

	url := fmt.Sprintf("%s/kots_metrics/preflights/%s/%s?%s", endpoint, appID, clusterID, urlValues.Encode())
	var buf bytes.Buffer
	postReq, err := util.NewRetryableRequest("POST", url, &buf)
	if err != nil {
		return errors.Wrap(err, "failed to call newrequest")
	}
	postReq.Header.Add("Authorization", license.GetLicenseID())
	postReq.Header.Set("Content-Type", "application/json")

	resp, err := util.DefaultHTTPClient.Do(postReq)
	if err != nil {
		return errors.Wrap(err, "failed to send preflights reports")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return errors.Errorf("Unexpected status code %d", resp.StatusCode)
	}
	return nil
}
