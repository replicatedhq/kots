package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	cron "github.com/robfig/cron/v3"
)

type SetAutomaticUpdatesConfigRequest struct {
	UpdateCheckerSpec string              `json:"updateCheckerSpec"`
	AutoDeploy        apptypes.AutoDeploy `json:"autoDeploy"`
}

type SetAutomaticUpdatesConfigResponse struct {
	Error string `json:"error"`
}

type GetAutomaticUpdatesConfigResponse struct {
	UpdateCheckerSpec string              `json:"updateCheckerSpec"`
	AutoDeploy        apptypes.AutoDeploy `json:"autoDeploy"`
	Error             string              `json:"error"`
}

func (h *Handler) SetAutomaticUpdatesConfig(w http.ResponseWriter, r *http.Request) {
	updateCheckerSpecResponse := &SetAutomaticUpdatesConfigResponse{}

	configureAutomaticUpdatesRequest := SetAutomaticUpdatesConfigRequest{}
	if err := json.NewDecoder(r.Body).Decode(&configureAutomaticUpdatesRequest); err != nil {
		updateCheckerSpecResponse.Error = "failed to decode request body"
		logger.Error(errors.Wrap(err, updateCheckerSpecResponse.Error))
		JSON(w, http.StatusBadRequest, updateCheckerSpecResponse)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		updateCheckerSpecResponse.Error = "failed to get app from slug"
		logger.Error(errors.Wrap(err, updateCheckerSpecResponse.Error))
		JSON(w, http.StatusInternalServerError, updateCheckerSpecResponse)
		return
	}

	license, err := kotsutil.LoadLicenseFromBytes([]byte(foundApp.License))
	if err != nil {
		updateCheckerSpecResponse.Error = "failed to get license from app"
		logger.Error(errors.Wrap(err, updateCheckerSpecResponse.Error))
		JSON(w, http.StatusInternalServerError, updateCheckerSpecResponse)
		return
	}

	var licenseChan *kotsv1beta1.Channel
	if foundApp.ChannelID == "" {
		// TODO: Backfill app.ChannelID in the database, this is an install from before multi-channel was introduced
		if licenseChan, err = kotsutil.FindChannelInLicense(license.Spec.ChannelID, license); err != nil {
			updateCheckerSpecResponse.Error = "failed to find channel in license"
			logger.Error(errors.Wrap(err, updateCheckerSpecResponse.Error))
			JSON(w, http.StatusInternalServerError, updateCheckerSpecResponse)
			return
		}
	} else {
		if licenseChan, err = kotsutil.FindChannelInLicense(foundApp.ChannelID, license); err != nil {
			updateCheckerSpecResponse.Error = "failed to find channel in license"
			logger.Error(errors.Wrap(err, updateCheckerSpecResponse.Error))
			JSON(w, http.StatusInternalServerError, updateCheckerSpecResponse)
			return
		}
	}

	// Check if the deploy update configuration is valid based on app channel
	if licenseChan.IsSemverRequired {
		if configureAutomaticUpdatesRequest.AutoDeploy == apptypes.AutoDeploySequence {
			updateCheckerSpecResponse.Error = "automatic updates based on sequence type are not supported for semantic versioning apps"
			JSON(w, http.StatusUnprocessableEntity, updateCheckerSpecResponse)
			return
		}
	} else {
		if configureAutomaticUpdatesRequest.AutoDeploy != apptypes.AutoDeployDisabled && configureAutomaticUpdatesRequest.AutoDeploy != apptypes.AutoDeploySequence {
			updateCheckerSpecResponse.Error = "automatic updates based on semantic versioning are not supported for non-semantic versioning apps"
			JSON(w, http.StatusUnprocessableEntity, updateCheckerSpecResponse)
			return
		}
	}

	if foundApp.IsAirgap {
		updateCheckerSpecResponse.Error = "airgap scheduled update checks are not supported"
		logger.Error(errors.New(updateCheckerSpecResponse.Error))
		JSON(w, http.StatusBadRequest, updateCheckerSpecResponse)
		return
	}

	// validate cron spec
	cronSpec := configureAutomaticUpdatesRequest.UpdateCheckerSpec
	if cronSpec != "@never" && cronSpec != "@default" {
		_, err := cron.ParseStandard(cronSpec)
		if err != nil {
			updateCheckerSpecResponse.Error = "failed to parse cron spec"
			logger.Error(errors.Wrap(err, updateCheckerSpecResponse.Error))
			JSON(w, http.StatusInternalServerError, updateCheckerSpecResponse)
			return
		}
	}

	if err := store.GetStore().SetUpdateCheckerSpec(foundApp.ID, cronSpec); err != nil {
		updateCheckerSpecResponse.Error = "failed to set update checker spec"
		logger.Error(errors.Wrap(err, updateCheckerSpecResponse.Error))
		JSON(w, http.StatusInternalServerError, updateCheckerSpecResponse)
		return
	}

	if err := store.GetStore().SetAutoDeploy(foundApp.ID, configureAutomaticUpdatesRequest.AutoDeploy); err != nil {
		updateCheckerSpecResponse.Error = "failed to set auto deploy"
		logger.Error(errors.Wrap(err, updateCheckerSpecResponse.Error))
		JSON(w, http.StatusInternalServerError, updateCheckerSpecResponse)
		return
	}

	// reconfigure update checker for the app
	if err := updatechecker.Configure(foundApp, cronSpec); err != nil {
		updateCheckerSpecResponse.Error = "failed to reconfigure update checker cron job"
		logger.Error(errors.Wrap(err, updateCheckerSpecResponse.Error))
		JSON(w, http.StatusInternalServerError, updateCheckerSpecResponse)
		return
	}

	JSON(w, http.StatusNoContent, "")
}

func (h *Handler) GetAutomaticUpdatesConfig(w http.ResponseWriter, r *http.Request) {
	getCheckerSpecResponse := &GetAutomaticUpdatesConfigResponse{}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		getCheckerSpecResponse.Error = "failed to get app from slug"
		logger.Error(errors.Wrap(err, getCheckerSpecResponse.Error))
		JSON(w, http.StatusInternalServerError, getCheckerSpecResponse)
		return
	}
	getCheckerSpecResponse.UpdateCheckerSpec = foundApp.UpdateCheckerSpec
	getCheckerSpecResponse.AutoDeploy = foundApp.AutoDeploy

	JSON(w, http.StatusOK, getCheckerSpecResponse)
}
