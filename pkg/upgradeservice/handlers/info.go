package handlers

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

type InfoResponse struct {
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
	HasPreflight   bool   `json:"hasPreflight"`
	IsConfigurable bool   `json:"isConfigurable"`
}

func (h *Handler) Info(w http.ResponseWriter, r *http.Request) {
	response := InfoResponse{
		Success: false,
	}

	params := GetContextParams(r)

	kotsKinds, err := kotsutil.LoadKotsKinds(params.AppArchive)
	if err != nil {
		response.Error = "failed to load kots kinds from path"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	response.Success = true
	response.HasPreflight = kotsKinds.HasPreflights()
	response.IsConfigurable = kotsKinds.IsConfigurable()

	JSON(w, http.StatusOK, response)
}
