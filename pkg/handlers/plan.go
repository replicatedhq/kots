package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/plan"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/store"
)

type UpdatePlanStepRequest struct {
	VersionLabel      string               `json:"versionLabel"`
	Status            types.PlanStepStatus `json:"status"`
	StatusDescription string               `json:"statusDescription"`
	Output            string               `json:"output"`
}

type UpdatePlanStepResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) UpdatePlanStep(w http.ResponseWriter, r *http.Request) {
	response := UpdatePlanStepResponse{
		Success: false,
	}

	request := UpdatePlanStepRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error = "failed to decode request body"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]
	stepID := mux.Vars(r)["stepID"]

	opts := plan.UpdateStepOptions{
		AppSlug:           appSlug,
		VersionLabel:      request.VersionLabel,
		StepID:            stepID,
		Status:            request.Status,
		StatusDescription: request.StatusDescription,
		Output:            request.Output,
	}
	if err := plan.UpdateStep(store.GetStore(), opts); err != nil {
		response.Error = "failed to update plan step"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	response.Success = true
	JSON(w, http.StatusOK, response)
}

type GetCurrentPlanStatusResponse struct {
	CurrentMessage string `json:"currentMessage"`
	Status         string `json:"status"`
}

func (h *Handler) GetCurrentPlanStatus(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(errors.Wrap(err, "failed to get app"))
		return
	}

	p, updatedAt, err := store.GetStore().GetCurrentPlan(a.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(errors.Wrap(err, "failed to get active plan"))
		return
	}
	if p == nil || time.Since(*updatedAt) > time.Minute {
		JSON(w, http.StatusOK, GetCurrentPlanStatusResponse{})
		return
	}

	stepType := r.URL.Query().Get("stepType")

	if stepType == "" {
		status, description := p.GetStatus()
		JSON(w, http.StatusOK, GetCurrentPlanStatusResponse{
			CurrentMessage: description,
			Status:         string(status),
		})
		return
	}

	for _, s := range p.Steps {
		if s.Type == types.PlanStepType(stepType) {
			JSON(w, http.StatusOK, GetCurrentPlanStatusResponse{
				CurrentMessage: s.StatusDescription,
				Status:         string(s.Status),
			})
			return
		}
	}
}
