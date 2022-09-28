package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	"github.com/replicatedhq/kots/pkg/util"
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

	if util.IsHelmManaged() {
		helmApp := helm.GetHelmApp(mux.Vars(r)["appSlug"])
		license, err := helm.GetChartLicenseFromSecretOrDownload(helmApp)
		if err != nil {
			logger.Error(errors.Wrap(err, updateCheckerSpecResponse.Error))
			JSON(w, http.StatusInternalServerError, updateCheckerSpecResponse)
			return
		}
		if license == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if helmApp.GetIsAirgap() {
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

		helm.SetUpdateCheckSpec(helmApp, cronSpec)

		// reconfigure update checker for the app
		if err := updatechecker.Configure(helmApp, cronSpec); err != nil {
			updateCheckerSpecResponse.Error = "failed to reconfigure update checker cron job"
			logger.Error(errors.Wrap(err, updateCheckerSpecResponse.Error))
			JSON(w, http.StatusInternalServerError, updateCheckerSpecResponse)
			return
		}

		JSON(w, http.StatusNoContent, "")
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

	// Check if the deploy update configuration is valid based on app channel
	if license.Spec.IsSemverRequired {
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

	if util.IsHelmManaged() {
		helmApp := helm.GetHelmApp(mux.Vars(r)["appSlug"])
		if helmApp == nil {
			JSON(w, http.StatusNotFound, getCheckerSpecResponse)
			return
		}

		spec, err := helm.GetUpdateCheckSpec(helmApp)
		if err != nil {
			getCheckerSpecResponse.Error = "failed to get schedule spec map"
			logger.Error(errors.Wrap(err, getCheckerSpecResponse.Error))
			JSON(w, http.StatusInternalServerError, getCheckerSpecResponse)
			return
		}

		getCheckerSpecResponse.UpdateCheckerSpec = spec
	} else {
		foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
		if err != nil {
			getCheckerSpecResponse.Error = "failed to get app from slug"
			logger.Error(errors.Wrap(err, getCheckerSpecResponse.Error))
			JSON(w, http.StatusInternalServerError, getCheckerSpecResponse)
			return
		}
		getCheckerSpecResponse.UpdateCheckerSpec = foundApp.UpdateCheckerSpec
		getCheckerSpecResponse.AutoDeploy = foundApp.AutoDeploy
	}

	JSON(w, http.StatusOK, getCheckerSpecResponse)
}
