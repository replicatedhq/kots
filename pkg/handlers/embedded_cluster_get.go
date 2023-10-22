package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

func (h *Handler) GetEmbeddedClusterNodes(w http.ResponseWriter, r *http.Request) {
	client, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	nodes, err := embeddedcluster.GetNodes(r.Context(), client)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, nodes)
}

func (h *Handler) GetEmbeddedClusterNode(w http.ResponseWriter, r *http.Request) {
	client, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	nodeName := mux.Vars(r)["nodeName"]
	node, err := embeddedcluster.GetNode(r.Context(), client, nodeName)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, node)
}
