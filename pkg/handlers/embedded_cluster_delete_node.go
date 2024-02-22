package handlers

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *Handler) DeleteEmbeddedClusterNode(w http.ResponseWriter, r *http.Request) {
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

	restconfig, err := k8sutil.GetClusterConfig()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	nodeName := mux.Vars(r)["nodeName"]
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Errorf("Failed to delete node %s: not found", nodeName)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := embeddedcluster.DeleteNode(ctx, client, restconfig, node); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	logger.Infof("Node %s successfully deleted", node.Name)
}
