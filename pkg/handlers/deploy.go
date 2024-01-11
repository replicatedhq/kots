package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/blang/semver"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/buildversion"
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

	sequence, err := strconv.ParseInt(mux.Vars(r)["sequence"], 10, 64)
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

	deployedSequence, err := store.GetStore().GetCurrentParentSequence(a.ID, downstreams[0].ClusterID)
	if err != nil {
		errMsg := "failed to get deployed sequence"
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}
	if sequence == deployedSequence {
		logger.Info(fmt.Sprintf("not deploying version %d because it's currently deployed", int64(sequence)))
		deployAppVersionResponse.Success = true
		JSON(w, http.StatusOK, deployAppVersionResponse)
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

	versions, err := store.GetStore().GetDownstreamVersions(a.ID, downstreams[0].ClusterID, true)
	if err != nil {
		errMsg := "failed to get app versions"
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}
	for _, v := range versions.PastVersions {
		if int64(sequence) == v.Sequence {
			// a past version is being deployed/rolled back to, disable automatic deployments so that it doesn't undo this action later
			logger.Infof("disabling automatic deployments because a past version is being deployed for app %s", a.Slug)
			if err := store.GetStore().SetAutoDeploy(a.ID, apptypes.AutoDeployDisabled); err != nil {
				logger.Error(errors.Wrap(err, "failed to set versioning auto deploy"))
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
		cause := errors.Cause(err)
		if _, ok := cause.(util.ActionableError); ok {
			deployAppVersionResponse.Error = cause.Error()
		} else {
			deployAppVersionResponse.Error = "failed to queue version for deployment"
		}
		logger.Error(errors.Wrap(err, "failed to queue version for deployment"))
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}

	// preflights reports
	go func() {
		if request.IsSkipPreflights || request.ContinueWithFailedPreflights {
			if err := reporting.WaitAndReportPreflightChecks(a.ID, int64(sequence), request.IsSkipPreflights, request.IsCLI); err != nil {
				logger.Debugf("failed to send preflights data to replicated app: %v", err)
				return
			}
		}
	}()

	deployAppVersionResponse.Success = true

	JSON(w, http.StatusOK, deployAppVersionResponse)
}

type AdminConsoleUpgradeError struct {
	IsCritical bool
	Message    string
}

func (e AdminConsoleUpgradeError) Error() string {
	return e.Message
}

func getKotsUpgradeVersion(kotsKinds *kotsutil.KotsKinds, latestVersion string) (string, error) {
	if !kotsutil.IsKotsAutoUpgradeSupported(&kotsKinds.KotsApplication) {
		return "", AdminConsoleUpgradeError{
			Message: "admin console auto updates feature flag not enabled",
		}
	}

	if kotsKinds.KotsApplication.Spec.MinKotsVersion == "" && kotsKinds.KotsApplication.Spec.TargetKotsVersion == "" {
		return "", AdminConsoleUpgradeError{
			Message: "no version requirement found in app",
		}
	}

	kotsUpgradeVersion := kotsKinds.KotsApplication.Spec.MinKotsVersion
	if kotsKinds.KotsApplication.Spec.TargetKotsVersion != "" {
		kotsUpgradeVersion = kotsKinds.KotsApplication.Spec.TargetKotsVersion
	} else if latestVersion != "" {
		kotsUpgradeVersion = latestVersion
	}

	upgradeSemver, err := semver.ParseTolerant(kotsUpgradeVersion)
	if err != nil {
		return "", AdminConsoleUpgradeError{
			Message:    errors.Wrapf(err, "failed to parse upgrade version %s", kotsUpgradeVersion).Error(),
			IsCritical: true,
		}
	}

	thisSemver, err := semver.ParseTolerant(buildversion.Version())
	if err != nil {
		return "", AdminConsoleUpgradeError{
			Message:    errors.Wrapf(err, "failed to parse this version %s", buildversion.Version()).Error(),
			IsCritical: true,
		}
	}

	if thisSemver.GTE(upgradeSemver) {
		return "", AdminConsoleUpgradeError{
			Message: "admin console is already at or above target version",
		}
	}

	return kotsUpgradeVersion, nil
}
