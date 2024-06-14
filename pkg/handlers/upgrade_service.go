package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	"github.com/replicatedhq/kots/pkg/upgradeservice"
	upgradeservicetypes "github.com/replicatedhq/kots/pkg/upgradeservice/types"
)

type StartUpgradeServiceRequest struct {
	KOTSVersion  string `json:"kotsVersion"`
	VersionLabel string `json:"versionLabel"`
	UpdateCursor string `json:"updateCursor"`
}

type StartUpgradeServiceResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) StartUpgradeService(w http.ResponseWriter, r *http.Request) {
	response := StartUpgradeServiceResponse{
		Success: false,
	}

	request := StartUpgradeServiceRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error = "failed to decode request body"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		response.Error = "failed to get app from app slug"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	license, err := kotsutil.LoadLicenseFromBytes([]byte(foundApp.License))
	if err != nil {
		response.Error = "failed to parse app license"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	updates, err := updatechecker.GetAvailableUpdates(store.GetStore(), foundApp, license)
	if err != nil {
		response.Error = "failed to get available updates"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	isDeployable, nonDeployableCause := false, "update not found"
	for _, u := range updates {
		if u.UpdateCursor == request.UpdateCursor {
			isDeployable, nonDeployableCause = u.IsDeployable, u.NonDeployableCause
			break
		}
	}
	if !isDeployable {
		response.Error = nonDeployableCause
		JSON(w, http.StatusBadRequest, response)
		return
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
	if err != nil {
		response.Error = "failed to get registry details for app"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	baseArchive, baseSequence, err := store.GetStore().GetAppVersionBaseArchive(foundApp.ID, request.VersionLabel)
	if err != nil {
		response.Error = "failed to get app version base archive"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	nextSequence, err := store.GetStore().GetNextAppSequence(foundApp.ID)
	if err != nil {
		response.Error = "failed to get next app sequence"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	err = upgradeservice.Start(upgradeservicetypes.UpgradeServiceParams{
		AppID:       foundApp.ID,
		AppSlug:     foundApp.Slug,
		AppIsAirgap: foundApp.IsAirgap,
		AppIsGitOps: foundApp.IsGitOps,
		AppLicense:  foundApp.License,

		BaseArchive:  baseArchive,
		BaseSequence: baseSequence,
		NextSequence: nextSequence,

		UpdateCursor: request.UpdateCursor,

		RegistryEndpoint:   registrySettings.Hostname,
		RegistryUsername:   registrySettings.Username,
		RegistryPassword:   registrySettings.Password,
		RegistryNamespace:  registrySettings.Namespace,
		RegistryIsReadOnly: registrySettings.IsReadOnly,

		ReportingInfo: reporting.GetReportingInfo(foundApp.ID),
	})
	if err != nil {
		response.Error = "failed to start upgrade service"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	response.Success = true

	JSON(w, http.StatusOK, response)
}
