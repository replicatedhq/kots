package handlers

import (
	"context"
	goerrors "errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/kotsadm/pkg/kurl"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func (h *Handler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.Clientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	restconfig, err := config.GetConfig()
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

	if err := kurl.DeleteNode(ctx, client, restconfig, node); err != nil {
		logger.Error(err)
		if goerrors.Is(err, kurl.ErrNoEkco) {
			w.WriteHeader(http.StatusUnprocessableEntity)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	logger.Infof("Node %s successfully deleted", node.Name)
}
