package reporting

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/segmentio/ksuid"
)

func SendAppInfo(appID string) error {
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

	postReq, err := http.NewRequest("POST", url, nil)
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

func GetReportingInfo(appID string) *types.ReportingInfo {
	r := types.ReportingInfo{
		InstanceID: appID,
	}

	configMap, err := k8sutil.GetKotsadmIDConfigMap()
	if err != nil {
		r.ClusterID = ksuid.New().String()
	} else if configMap != nil {
		r.ClusterID = configMap.Data["id"]
	} else {
		// configmap is missing for some reason, recreate with new guid, this will appear as a new instance in the report
		r.ClusterID = ksuid.New().String()
		k8sutil.CreateKotsadmIDConfigMap(r.ClusterID)
	}

	di, err := getDownstreamInfo(appID)
	if err != nil {
		logger.Debugf("failed to get downstream info: %v", err.Error())
	}
	if di != nil {
		r.Downstream = *di
	}

	// get kubernetes cluster version
	k8sVersion, err := k8sutil.GetK8sVersion()
	if err != nil {
		logger.Debugf("failed to get k8s version: %v", err.Error())
	} else {
		r.K8sVersion = k8sVersion
	}

	// get app status
	appStatus, err := store.GetStore().GetAppStatus(appID)
	if err != nil {
		logger.Debugf("failed to get app status: %v", err.Error())
	} else {
		r.AppStatus = string(appStatus.State)
	}

	// check if embedded cluster
	r.IsKurl = kurl.IsKurl()

	return &r
}

func getDownstreamInfo(appID string) (*types.DownstreamInfo, error) {
	di := types.DownstreamInfo{}

	downstreams, err := store.GetStore().ListDownstreamsForApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}
	if len(downstreams) == 0 {
		return nil, errors.New("no downstreams found for app")
	}

	deployedAppSequence, err := store.GetStore().GetCurrentParentSequence(appID, downstreams[0].ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current downstream parent sequence")
	}

	// info about the deployed app sequence
	if deployedAppSequence != -1 {
		deployedArchiveDir, err := ioutil.TempDir("", "kotsadm")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(deployedArchiveDir)

		err = store.GetStore().GetAppVersionArchive(appID, deployedAppSequence, deployedArchiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get app version archive")
		}

		deployedKotsKinds, err := kotsutil.LoadKotsKindsFromPath(deployedArchiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load kotskinds from path")
		}

		di.Cursor = deployedKotsKinds.Installation.Spec.UpdateCursor
		di.ChannelID = deployedKotsKinds.Installation.Spec.ChannelID
		di.ChannelName = deployedKotsKinds.Installation.Spec.ChannelName

		if deployedKotsKinds.HelmChart != nil {
			di.IsNativeHelm = deployedKotsKinds.HelmChart.Spec.UseHelmInstall
		}
	}

	return &di, nil
}
