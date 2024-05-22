package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/upgrader"
	upgradertypes "github.com/replicatedhq/kots/pkg/upgrader/types"
)

type StartUpgraderRequest struct {
	KOTSVersion string `json:"kotsVersion"`
}

type StartUpgraderResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) StartUpgrader(w http.ResponseWriter, r *http.Request) {
	response := StartUpgraderResponse{
		Success: false,
	}

	request := StartUpgraderRequest{}
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

	// TODO NOW: get version label from request
	baseArchive, baseSequence, err := store.GetStore().GetAppVersionBaseArchive(foundApp.ID, airgap.Spec.VersionLabel)
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

	// TODO NOW: get latest license? if done here, remove the one in upgrader bootstrap

	updateCursor, err := store.GetStore().GetCurrentUpdateCursor(foundApp.ID, latestLicense.Spec.ChannelID)
	if err != nil {
		response.Error = "failed to get current update cursor"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	// TODO NOW: get cursor from request
	// TODO NOW: download archive from replicated.app in online mode
	// TODO NOW: extract archive in airgap mode

	err = upgrader.Start(upgradertypes.StartOptions{
		KOTSVersion:      request.KOTSVersion,
		App:              foundApp,
		BaseArchive:      baseArchive,
		BaseSequence:     baseSequence,
		NextSequence:     nextSequence,
		UpdateCursor:     updateCursor,
		RegistrySettings: registrySettings,
		// TODO NOW: reporting info
	})
	if err != nil {
		response.Error = "failed to start upgrader"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	response.Success = true

	JSON(w, http.StatusOK, response)
}
