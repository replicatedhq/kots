package handlers

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/pkg/helmvm"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *Handler) DrainHelmVMNode(w http.ResponseWriter, r *http.Request) {
	client, err := k8sutil.GetClientset()
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
			logger.Errorf("Failed to drain node %s: not found", nodeName)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// This pod may get evicted and not be able to respond to the request
	go func() {
		if err := helmvm.DrainNode(ctx, client, node); err != nil {
			logger.Error(err)
			return
		}
		logger.Infof("Node %s successfully drained", node.Name)
	}()
}
