package handlers

import (
	"encoding/json"
	"net/http"

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
