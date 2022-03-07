package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/replicatedhq/kots/pkg/kotsutil"

	"github.com/gorilla/mux"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	cron "github.com/robfig/cron/v3"
)

type ConfigureAutomaticUpdatesRequest struct {
	UpdateCheckerSpec string              `json:"updateCheckerSpec"`
	AutoDeploy        apptypes.AutoDeploy `json:"autoDeploy"`
}

type ConfigureAutomaticUpdatesResponse struct {
	Error string `json:"error"`
}

func (h *Handler) ConfigureAutomaticUpdates(w http.ResponseWriter, r *http.Request) {
	updateCheckerSpecResponse := &ConfigureAutomaticUpdatesResponse{}

	configureAutomaticUpdatesRequest := ConfigureAutomaticUpdatesRequest{}
	if err := json.NewDecoder(r.Body).Decode(&configureAutomaticUpdatesRequest); err != nil {
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

	license, err := kotsutil.LoadLicenseFromBytes([]byte(foundApp.License))
	if err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to get license from app"
		JSON(w, 500, updateCheckerSpecResponse)
		return
	}

	// Check if the deploy update configuration is valid based on app channel
	if license.Spec.IsSemverRequired {
		if configureAutomaticUpdatesRequest.AutoDeploy == apptypes.AutoDeploySequence {
			updateCheckerSpecResponse.Error = "automatic updates based on sequence type are not supported for semantic versioning apps"
			JSON(w, 422, updateCheckerSpecResponse)
			return
		}
	} else {
		if configureAutomaticUpdatesRequest.AutoDeploy != apptypes.AutoDeployDisabled && configureAutomaticUpdatesRequest.AutoDeploy != apptypes.AutoDeploySequence {
			updateCheckerSpecResponse.Error = "automatic updates based on semantic versioning are not supported for non-semantic versioning apps"
			JSON(w, 422, updateCheckerSpecResponse)
			return
		}
	}

	if foundApp.IsAirgap {
		logger.Error(errors.New("airgap scheduled update checks are not supported"))
		updateCheckerSpecResponse.Error = "airgap scheduled update checks are not supported"
		JSON(w, 400, updateCheckerSpecResponse)
		return
	}

	// validate cron spec
	cronSpec := configureAutomaticUpdatesRequest.UpdateCheckerSpec
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

	if err := store.GetStore().SetAutoDeploy(foundApp.ID, configureAutomaticUpdatesRequest.AutoDeploy); err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to set auto deploy"
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

	JSON(w, 204, "")
}
