package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/autodeployer"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	cron "github.com/robfig/cron/v3"
)

type UpdateCheckerSpecRequest struct {
	UpdateCheckerSpec        string                    `json:"updateCheckerSpec"`
	SemverAutoDeploy         apptypes.SemverAutoDeploy `json:"semverAutoDeploy"`
	SemverAutoDeploySchedule string                    `json:"semverAutoDeploySchedule"`
}

type UpdateCheckerSpecResponse struct {
	Error string `json:"error"`
}

func (h *Handler) ConfigureAutomaticUpdates(w http.ResponseWriter, r *http.Request) {
	updateCheckerSpecResponse := &UpdateCheckerSpecResponse{}

	updateCheckerSpecRequest := UpdateCheckerSpecRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateCheckerSpecRequest); err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to decode request body"
		JSON(w, 400, updateCheckerSpecResponse)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to get app from slug"
		JSON(w, 500, updateCheckerSpecResponse)
		return
	}

	if foundApp.IsAirgap {
		logger.Error(errors.New("airgap scheduled update checks are not supported"))
		updateCheckerSpecResponse.Error = "airgap scheduled update checks are not supported"
		JSON(w, 400, updateCheckerSpecResponse)
		return
	}

	// validate cron spec
	cronSpec := updateCheckerSpecRequest.UpdateCheckerSpec
	if cronSpec != "@never" && cronSpec != "@default" {
		_, err := cron.ParseStandard(cronSpec)
		if err != nil {
			logger.Error(err)
			updateCheckerSpecResponse.Error = "failed to parse cron spec"
			JSON(w, 500, updateCheckerSpecResponse)
			return
		}
	}

	if err := store.GetStore().SetUpdateCheckerSpec(foundApp.ID, cronSpec); err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to set update checker spec"
		JSON(w, 500, updateCheckerSpecResponse)
		return
	}

	if err := store.GetStore().SetSemverAutoDeploy(foundApp.ID, updateCheckerSpecRequest.SemverAutoDeploy, updateCheckerSpecRequest.SemverAutoDeploySchedule); err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to set semver auto deploy"
		JSON(w, 500, updateCheckerSpecResponse)
		return
	}

	// reconfigure update checker for the app
	if err := updatechecker.Configure(foundApp.ID); err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to reconfigure update checker cron job"
		JSON(w, 500, updateCheckerSpecResponse)
		return
	}

	// reconfigure auto deployer for the app
	if err := autodeployer.Configure(foundApp.ID); err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to reconfigure auto deployer cron job"
		JSON(w, 500, updateCheckerSpecResponse)
		return
	}

	JSON(w, 204, "")
}
