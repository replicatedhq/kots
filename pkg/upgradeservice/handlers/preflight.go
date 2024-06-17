package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	upgradepreflight "github.com/replicatedhq/kots/pkg/upgradeservice/preflight"
)

type GetPreflightResultResponse struct {
	PreflightProgress string                         `json:"preflightProgress,omitempty"`
	PreflightResult   preflighttypes.PreflightResult `json:"preflightResult"`
}

func (h *Handler) GetPreflightResult(w http.ResponseWriter, r *http.Request) {
	params := GetContextParams(r)
	appSlug := mux.Vars(r)["appSlug"]

	if params.AppSlug != appSlug {
		logger.Error(errors.Errorf("app slug in path %s does not match app slug in context %s", appSlug, params.AppSlug))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	preflightData, err := upgradepreflight.GetPreflightData()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get preflight data"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := GetPreflightResultResponse{
		PreflightResult:   *preflightData.Result,
		PreflightProgress: preflightData.Progress,
	}
	JSON(w, 200, response)
}
