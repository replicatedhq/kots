package handlers

import (
	"net/http"

	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

type GetClustersResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func GetClusters(w http.ResponseWriter, r *http.Request) {
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

	clusters, err := store.GetStore().ListClusters()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	resp := []GetClustersResponse{}
	for idx, val := range clusters {
		resp = append(resp, GetClustersResponse{
			ID:   idx,
			Name: val,
		})
	}

	JSON(w, 200, resp)
}
