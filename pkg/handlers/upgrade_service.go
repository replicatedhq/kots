package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
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

	// TODO NOW: required releases

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

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
	if err != nil {
		response.Error = "failed to get registry details for app"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	// TODO NOW: send version label in request
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

	// TODO NOW: send cursor in request
	err = upgradeservice.Start(upgradeservicetypes.StartOptions{
		KOTSVersion:      request.KOTSVersion,
		App:              foundApp,
		BaseArchive:      baseArchive,
		BaseSequence:     baseSequence,
		NextSequence:     nextSequence,
		UpdateCursor:     request.UpdateCursor,
		RegistrySettings: registrySettings,
		// TODO NOW: reporting info
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
