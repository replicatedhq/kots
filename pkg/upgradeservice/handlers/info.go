package handlers

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
)

type InfoResponse struct {
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
	HasPreflight   bool   `json:"hasPreflight"`
	IsConfigurable bool   `json:"isConfigurable"`
	IsEC2Install   bool   `json:"isEC2Install"`
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
	response.IsEC2Install = util.IsEC2Install()

	JSON(w, http.StatusOK, response)
}
