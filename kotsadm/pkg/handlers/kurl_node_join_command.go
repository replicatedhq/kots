package handlers

import (
	"net/http"
	"time"

	"github.com/replicatedhq/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kotsadm/pkg/kurl"
	"github.com/replicatedhq/kotsadm/pkg/logger"
)

type GenerateNodeJoinCommandResponse struct {
	Command []string `json:"command"`
	Expiry  string   `json:"expiry"`
}

func GenerateNodeJoinCommandWorker(w http.ResponseWriter, r *http.Request) {
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

	client, err := k8s.Clientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	command, expiry, err := kurl.GenerateAddNodeCommand(client, false)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, GenerateNodeJoinCommandResponse{
		Command: command,
		Expiry:  expiry.Format(time.RFC3339),
	})
}

func GenerateNodeJoinCommandMaster(w http.ResponseWriter, r *http.Request) {
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

	client, err := k8s.Clientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	command, expiry, err := kurl.GenerateAddNodeCommand(client, true)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, GenerateNodeJoinCommandResponse{
		Command: command,
		Expiry:  expiry.Format(time.RFC3339),
	})
}
