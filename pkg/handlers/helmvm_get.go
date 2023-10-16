package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/pkg/helmvm"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

func (h *Handler) GetHelmVMNodes(w http.ResponseWriter, r *http.Request) {
	client, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	nodes, err := helmvm.GetNodes(r.Context(), client)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, nodes)
}

func (h *Handler) GetHelmVMNode(w http.ResponseWriter, r *http.Request) {
	client, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	nodeName := mux.Vars(r)["nodeName"]
	node, err := helmvm.GetNode(r.Context(), client, nodeName)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, node)
}
