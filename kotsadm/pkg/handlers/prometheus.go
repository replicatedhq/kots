package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
)

type SetPrometheusAddressRequest struct {
	Value string `json:"value"`
}

func (h *Handler) SetPrometheusAddress(w http.ResponseWriter, r *http.Request) {
	setPrometheusAddressRequest := SetPrometheusAddressRequest{}
	if err := json.NewDecoder(r.Body).Decode(&setPrometheusAddressRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := store.GetStore().SetPrometheusAddress(setPrometheusAddressRequest.Value); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	JSON(w, 204, "")
}
