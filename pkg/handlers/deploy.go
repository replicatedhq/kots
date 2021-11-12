package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
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

func (h *Handler) DeployAppVersion(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	request := DeployAppVersionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.Error(errors.Wrap(err, "failed to decode request body"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse sequence number"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get app for slug %s", appSlug))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list downstreams for app"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if len(downstreams) == 0 {
		logger.Error(errors.Errorf("no downstreams for app %s", appSlug))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	status, err := store.GetStore().GetStatusForVersion(a.ID, downstreams[0].ClusterID, int64(sequence))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get update downstream status"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if status == storetypes.VersionPendingConfig {
		logger.Error(errors.Errorf("not deploying version %d because it's %s", int64(sequence), status))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := store.GetStore().DeleteDownstreamDeployStatus(a.ID, downstreams[0].ClusterID, int64(sequence)); err != nil {
		logger.Error(errors.Wrap(err, "failed to delete downstream deploy status"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := version.DeployVersion(a.ID, int64(sequence)); err != nil {
		logger.Error(errors.Wrap(err, "failed to queue version for deployment"))
		w.WriteHeader(http.StatusInternalServerError)
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

	JSON(w, 204, "")
}
