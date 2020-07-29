package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/gitops"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
)

type UpdateAppGitOpsRequest struct {
	GitOpsInput GitOpsInput `json:"gitOpsInput"`
}
type GitOpsInput struct {
	Uri    string `json:"uri"`
	Branch string `json:"branch"`
	Path   string `json:"path"`
	Format string `json:"format"`
	Action string `json:"action"`
}

func UpdateAppGitOps(w http.ResponseWriter, r *http.Request) {
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

	updateAppGitOpsRequest := UpdateAppGitOpsRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateAppGitOpsRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	appID := mux.Vars(r)["appId"]
	clusterID := mux.Vars(r)["clusterId"]

	a, err := app.Get(appID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gitOpsInput := updateAppGitOpsRequest.GitOpsInput
	if err := gitops.UpdateDownstreamGitOps(a.ID, clusterID, gitOpsInput.Uri, gitOpsInput.Branch, gitOpsInput.Path, gitOpsInput.Format, gitOpsInput.Action); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusNoContent, "")
}

func DisableAppGitOps(w http.ResponseWriter, r *http.Request) {
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

	appID := mux.Vars(r)["appId"]
	clusterID := mux.Vars(r)["clusterId"]

	a, err := app.Get(appID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	downstreamGitOps, err := gitops.GetDownstreamGitOps(a.ID, clusterID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if downstreamGitOps != nil {
		err := gitops.DisableDownstreamGitOps(a.ID, clusterID)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	JSON(w, http.StatusNoContent, "")
}
