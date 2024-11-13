package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
)

type GetEmbeddedClusterRolesResponse struct {
	Roles []string `json:"roles"`
}

func (h *Handler) GetEmbeddedClusterNodes(w http.ResponseWriter, r *http.Request) {
	if !util.IsEmbeddedCluster() {
		logger.Errorf("not an embedded cluster")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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
	if !util.IsEmbeddedCluster() {
		logger.Errorf("not an embedded cluster")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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

func (h *Handler) GetEmbeddedClusterRoles(w http.ResponseWriter, r *http.Request) {
	if !util.IsEmbeddedCluster() {
		logger.Errorf("not an embedded cluster")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	kbClient, err := h.GetKubeClient(r.Context())
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	roles, err := embeddedcluster.GetRoles(r.Context(), kbClient)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, GetEmbeddedClusterRolesResponse{Roles: roles})
}
