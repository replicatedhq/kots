package reporting

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
)

var onlineAppInfoMtx sync.Mutex

func (r *OnlineReporter) SubmitAppInfo(appID string) error {
	// make sure events are reported in order
	onlineAppInfoMtx.Lock()
	defer func() {
		time.Sleep(1 * time.Second)
		onlineAppInfoMtx.Unlock()
	}()

	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		if store.GetStore().IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get license for app")
	}

	license, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get license for app")
	}

	endpoint := license.Spec.Endpoint
	if !canReport(endpoint) {
		return nil
	}

	url := fmt.Sprintf("%s/kots_metrics/license_instance/info", endpoint)

	postReq, err := util.NewRequest("POST", url, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create http request")
	}
	postReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID)))))
	postReq.Header.Set("Content-Type", "application/json")

	reportingInfo := GetReportingInfo(a.ID)
	InjectReportingInfoHeaders(postReq, reportingInfo)

	resp, err := http.DefaultClient.Do(postReq)
	if err != nil {
		return errors.Wrap(err, "failed to post request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	return nil
}
