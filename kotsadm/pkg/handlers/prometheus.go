package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/replicatedhq/kots/kotsadm/pkg/kotsadmparams"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
)

type SetPrometheusAddressRequest struct {
	Value string `json:"value"`
}

func SetPrometheusAddress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	setPrometheusAddressRequest := SetPrometheusAddressRequest{}
	if err := json.NewDecoder(r.Body).Decode(&setPrometheusAddressRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := kotsadmparams.Set("PROMETHEUS_ADDRESS", setPrometheusAddressRequest.Value); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	JSON(w, 204, "")
}
