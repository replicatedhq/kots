package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/version"
)

type DeployAppVersionRequest struct {
	IsSkipPreflights             bool `json:"isSkipPreflights"`
	ContinueWithFailedPreflights bool `json:"continueWithFailedPreflights"`
	IsCLI                        bool `json:"isCli"`
}

type DeployAppVersionResponse struct {
	Success                 bool   `json:"success"`
	Error                   string `json:"error,omitempty"`
	IncompatibleKotsVersion bool   `json:"incompatibleKotsVersion,omitempty"`
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
		errMsg := "failed to get update downstream status"
		logger.Error(errors.Wrap(err, errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}

	if status == storetypes.VersionPendingConfig {
		errMsg := fmt.Sprintf("not deploying version %d because it's %s", int64(sequence), status)
		logger.Error(errors.New(errMsg))
		deployAppVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, deployAppVersionResponse)
		return
	}

	versions, err := store.GetStore().GetAppVersions(a.ID, downstreams[0].ClusterID)
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

	JSON(w, 204, deployAppVersionResponse)
}
