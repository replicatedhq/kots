package reporting

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func (r *OnlineReporter) SubmitPreflightData(license *kotsv1beta1.License, appID string, clusterID string, sequence int64, skipPreflights bool, installStatus storetypes.DownstreamVersionStatus, isCLI bool, preflightStatus string, appStatus string) error {
	endpoint := license.Spec.Endpoint
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
	postReq, err := util.NewRequest("POST", url, &buf)
	if err != nil {
		return errors.Wrap(err, "failed to call newrequest")
	}
	postReq.Header.Add("Authorization", license.Spec.LicenseID)
	postReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(postReq)
	if err != nil {
		return errors.Wrap(err, "failed to send preflights reports")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return errors.Errorf("Unexpected status code %d", resp.StatusCode)
	}
	return nil
}
