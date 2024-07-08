package handlers

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

type GetAppResponse struct {
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
	KOTSVersion    string `json:"kotsVersion"`
	HasPreflight   bool   `json:"hasPreflight"`
	IsConfigurable bool   `json:"isConfigurable"`
}

func (h *Handler) GetApp(w http.ResponseWriter, r *http.Request) {
	response := GetAppResponse{
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
	response.KOTSVersion = params.UpdateKOTSVersion
	response.HasPreflight = kotsKinds.HasPreflights()
	response.IsConfigurable = kotsKinds.IsConfigurable()

	JSON(w, http.StatusOK, response)
}
