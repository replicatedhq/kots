package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/blang/semver"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
)

type DeployAppVersionRequest struct {
	IsSkipPreflights             bool `json:"isSkipPreflights"`
	ContinueWithFailedPreflights bool `json:"continueWithFailedPreflights"`
	IsCLI                        bool `json:"isCli"`
}

type DeployAppVersionResponse struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) DeployAppVersion(w http.ResponseWriter, r *http.Request) {
	deployAppVersionResponse := DeployAppVersionResponse{
		Success: false,
	}

	appSlug := mux.Vars(r)["appSlug"]

	request := DeployAppVersionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		errMsg := "failed to decode request body"
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusBadRequest, deployAppVersionResponse)
		return
	}

	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		errMsg := "failed to parse sequence number"
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusBadRequest, deployAppVersionResponse)
		return
	}

	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get app for slug %s", appSlug)
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		errMsg := "failed to list downstreams for app"
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	} else if len(downstreams) == 0 {
		errMsg := fmt.Sprintf("no downstreams for app %s", appSlug)
		logger.Error(errors.New(errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}

	status, err := store.GetStore().GetStatusForVersion(a.ID, downstreams[0].ClusterID, int64(sequence))
	if err != nil {
		errMsg := fmt.Sprintf("failed to get status for version %d", sequence)
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}

	if status == storetypes.VersionPendingDownload || status == storetypes.VersionPendingConfig {
		errMsg := fmt.Sprintf("not deploying version %d because it's %s", int64(sequence), status)
		logger.Error(errors.New(errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusBadRequest, deployAppVersionResponse)
		return
	}

	kotsUpgradeNeeded, targetVersion, err := isKotsUpgradeNeeded(a, int64(sequence))
	if err != nil {
		errMsg := "failed to check if kots upgrade is needed"
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}

	if kotsUpgradeNeeded {
		logger.Debugf("will upgrade to Admin Console to version %s", targetVersion)

		deployAppVersionResponse.Status = "kots-upgrade-needed"
		JSON(w, http.StatusOK, deployAppVersionResponse)
		go func() {
			err := kotsadm.UpdateToVersion(targetVersion)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to start kots upgrade"))
			}
		}()
		return
	}

	versions, err := store.GetStore().GetAppVersions(a.ID, downstreams[0].ClusterID, true)
	if err != nil {
		errMsg := "failed to get app versions"
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}
	for _, v := range versions.PastVersions {
		if int64(sequence) == v.Sequence {
			// a past version is being deployed/rolled back to, disable semver automatic deployments so that it doesn't undo this action later
			logger.Infof("disabling semver automatic deployments because a past version is being deployed for app %s", a.Slug)
			if err := store.GetStore().SetSemverAutoDeploy(a.ID, apptypes.SemverAutoDeployDisabled); err != nil {
				logger.Error(errors.Wrap(err, "failed to set semver auto deploy"))
			}
			break
		}
	}

	if err := store.GetStore().DeleteDownstreamDeployStatus(a.ID, downstreams[0].ClusterID, int64(sequence)); err != nil {
		errMsg := "failed to delete downstream deploy status"
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}

	if err := version.DeployVersion(a.ID, int64(sequence)); err != nil {
		errMsg := "failed to queue version for deployment"
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}

	// preflights reports
	go func() {
		if request.IsSkipPreflights || request.ContinueWithFailedPreflights {
			if err := reporting.ReportAppInfo(a.ID, int64(sequence), request.IsSkipPreflights, request.IsCLI); err != nil {
				logger.Debugf("failed to send preflights data to replicated app: %v", err)
				return
			}
		}
	}()

	deployAppVersionResponse.Success = true

	JSON(w, http.StatusOK, deployAppVersionResponse)
}

func isKotsUpgradeNeeded(app *apptypes.App, sequence int64) (bool, string, error) {
	archivePath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return false, "", errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archivePath)

	err = store.GetStore().GetAppVersionArchive(app.ID, int64(sequence), archivePath)
	if err != nil {
		return false, "", errors.Wrapf(err, "failed to get archive for sequence %d", sequence)
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archivePath)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to load kots kinds")
	}

	latestVersion, _ := findLatestKotsVersion(app.ID, kotsKinds.License)

	targetVersion, err := getTargetKotsVersion(kotsKinds, latestVersion)
	if err != nil {
		if err, ok := err.(AdminConsoleUpgradeError); ok {
			if err.IsCritical {
				return false, "", errors.Wrap(err, "critical error while checking target kots version")
			}
			logger.Infof("skipping kots upgrade because %v", err)
			return false, "", nil
		}
		return false, "", errors.Wrap(err, "failed to find target version")
	}

	return true, targetVersion, nil
}

func findLatestKotsVersion(appID string, license *kotsv1beta1.License) (string, error) {
	url := fmt.Sprintf("%s/admin-console/version/latest", license.Spec.Endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create new request")
	}

	reportingInfo := reporting.GetReportingInfo(appID)
	reporting.InjectReportingInfoHeaders(req, reportingInfo)

	req.Header.Add("User-Agent", fmt.Sprintf("KOTS/%s", buildversion.Version()))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID)))))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute get request")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode >= 400 {
		if len(body) > 0 {
			return "", util.ActionableError{Message: string(body)}
		}
		return "", errors.Errorf("unexpected result from get request: %d", resp.StatusCode)
	}

	var versionInfo struct {
		Tag string `json:"tag"`
	}
	if err := json.Unmarshal(body, &versionInfo); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal response: %s", body)
	}

	return versionInfo.Tag, nil
}

type AdminConsoleUpgradeError struct {
	IsCritical bool
	Message    string
}

func (e AdminConsoleUpgradeError) Error() string {
	return e.Message
}

func getTargetKotsVersion(kotsKinds *kotsutil.KotsKinds, latestVersion string) (string, error) {
	if !kotsutil.IsKotsAutoUpgradeSupported(kotsKinds.KotsApplication) {
		return "", AdminConsoleUpgradeError{
			Message: "admin console auto updates feature flag not enabled",
		}
	}

	if kotsKinds.KotsApplication.Spec.MinKotsVersion == "" && kotsKinds.KotsApplication.Spec.TargetKotsVersion == "" {
		return "", AdminConsoleUpgradeError{
			Message: "no version requirement found in app",
		}
	}

	targetKotsVersion := kotsKinds.KotsApplication.Spec.MinKotsVersion
	if kotsKinds.KotsApplication.Spec.TargetKotsVersion != "" {
		targetKotsVersion = kotsKinds.KotsApplication.Spec.TargetKotsVersion
	} else if latestVersion != "" {
		targetKotsVersion = latestVersion
	}

	targetSemver, err := semver.ParseTolerant(targetKotsVersion)
	if err != nil {
		return "", AdminConsoleUpgradeError{
			Message:    errors.Wrapf(err, "failed to parse target version %s", targetKotsVersion).Error(),
			IsCritical: true,
		}
	}

	thisSemver, err := semver.ParseTolerant(buildversion.Version())
	if err != nil {
		return "", AdminConsoleUpgradeError{
			Message:    errors.Wrapf(err, "failed to parse this version %s", targetKotsVersion).Error(),
			IsCritical: true,
		}
	}

	if thisSemver.GTE(targetSemver) {
		return "", AdminConsoleUpgradeError{
			Message: errors.New("admdin console is already at or above target version").Error(),
		}
	}

	return targetKotsVersion, nil
}
