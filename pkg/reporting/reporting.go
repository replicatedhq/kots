package reporting

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	downstream "github.com/replicatedhq/kots/pkg/kotsadmdownstream"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
)

func isDevEnvironment() bool {
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		return true
	}
	return false
}

func SendPreflightsReportToReplicatedApp(license *kotsv1beta1.License, appID string, clusterID string, sequence int, skipPreflights bool, installStatus string) error {

	if isDevEnvironment() {
		return nil
	}

	urlValues := url.Values{}

	sequenceToStr := fmt.Sprintf("%d", sequence)
	skipPreflightsToStr := fmt.Sprintf("%t", skipPreflights)

	urlValues.Set("sequence", sequenceToStr)
	urlValues.Set("skipPreflights", skipPreflightsToStr)
	urlValues.Set("installStatus", installStatus)

	url := fmt.Sprintf("%s/kots_metrics/preflights/%s/%s?%s", license.Spec.Endpoint, appID, clusterID, urlValues.Encode())

	var buf bytes.Buffer
	postReq, err := http.NewRequest("POST", url, &buf)
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

func SendPreflightInfo(appID string, sequence int, isSkipPreflights bool, isUpdate bool) error {
	license, err := store.GetStore().GetLatestLicenseForApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to find license for app")
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to list downstreams for app")
	} else if len(downstreams) == 0 {
		err = errors.New("no downstreams for app")
		return err
	}

	clusterID := downstreams[0].ClusterID

	if isSkipPreflights || isUpdate {
		// at this point current version status does not exist so it's
		// neccessary to create thread to get it after version is deployed

		// isUpdate means that it's not on initial install
		go func() {
			<-time.After(20 * time.Second)
			currentVersion, err := downstream.GetCurrentVersion(appID, clusterID)
			if err != nil {
				logger.Debugf("failed to get current downstream version: %v", err)
				return
			}
			if currentVersion.Status != "" && currentVersion.Status != "deploying" {
				if err := SendPreflightsReportToReplicatedApp(license, appID, clusterID, sequence, isSkipPreflights, currentVersion.Status); err != nil {
					logger.Debugf("failed to send preflights data to replicated app: %v", err)
					return
				}
			}
		}()
	} else {
		if err := SendPreflightsReportToReplicatedApp(license, appID, clusterID, sequence, isSkipPreflights, ""); err != nil {
			return errors.Wrap(err, "failed to send preflights data to replicated app")
		}
	}

	return nil
}
