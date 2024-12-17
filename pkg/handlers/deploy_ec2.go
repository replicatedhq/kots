package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/plan"
	"github.com/replicatedhq/kots/pkg/store"
	upgradeservicetask "github.com/replicatedhq/kots/pkg/upgradeservice/task"
)

type DeployEC2AppVersionRequest struct {
	VersionLabel string `json:"versionLabel"`
	UpdateCursor string `json:"updateCursor"`
	ChannelID    string `json:"channelId"`
}

type DeployEC2AppVersionResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) DeployEC2AppVersion(w http.ResponseWriter, r *http.Request) {
	response := DeployEC2AppVersionResponse{
		Success: false,
	}

	request := DeployEC2AppVersionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error = "failed to decode request body"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]

	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		response.Error = "failed to get app from slug"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	// TODO (@salah): implement canStartUpgradeService logic here

	p, err := plan.PlanUpgrade(store.GetStore(), plan.PlanUpgradeOptions{
		AppSlug:      appSlug,
		VersionLabel: request.VersionLabel,
		UpdateCursor: request.UpdateCursor,
		ChannelID:    request.ChannelID,
	})
	if err != nil {
		response.Error = "failed to plan upgrade"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if err := store.GetStore().UpsertPlan(a.ID, request.VersionLabel, p); err != nil {
		response.Error = "failed to create plan"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	// TODO NOW: is this step needed? use plan step status instead? upgrade service can update status via api
	if err := upgradeservicetask.SetStatusStarting(appSlug, "Preparing..."); err != nil {
		response.Error = "failed to set task status"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	go func() {
		if err := plan.Execute(store.GetStore(), p); err != nil {
			logger.Error(errors.Wrap(err, "failed to execute upgrade plan"))
		}
	}()

	response.Success = true

	JSON(w, http.StatusOK, response)
}
