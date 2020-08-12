package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/gitops"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
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
		w.WriteHeader(http.StatusInternalServerError)
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

func InitGitOpsConnection(w http.ResponseWriter, r *http.Request) {
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

	d, err := downstream.Get(clusterID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	downstreamGitOps, err := gitops.GetDownstreamGitOps(a.ID, d.ClusterID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if downstreamGitOps == nil {
		logger.Error(errors.New("downstream gitops not found"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if downstreamGitOps.Format != "single" {
		logger.Error(errors.New("unsupported gitops format"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := gitops.TestGitOpsConnection(downstreamGitOps); err != nil {
		logger.Error(err)
		err = gitops.SetGitOpsError(a.ID, d.ClusterID, err.Error())
		if err != nil {
			logger.Error(err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := gitops.SetGitOpsError(a.ID, d.ClusterID, ""); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go func(appID, clusterID, appSlug, appName, downstreamName string) {
		currentVersion, err := downstream.GetCurrentVersion(appID, clusterID)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		pendingVersions, err := downstream.GetPendingVersions(appID, clusterID)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Create git commit for current version
		currentVersionArchive, err := version.GetAppVersionArchive(appID, currentVersion.ParentSequence)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(currentVersionArchive)

		_, err = gitops.CreateGitOpsCommit(downstreamGitOps, appSlug, appName, int(currentVersion.ParentSequence), currentVersionArchive, downstreamName)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Sort pending versions ascending before creating commits
		sort.Slice(pendingVersions, func(i, j int) bool {
			return pendingVersions[i].ParentSequence < pendingVersions[j].ParentSequence
		})
		// Create git commits for sorted pending versions
		for _, pendingVersion := range pendingVersions {
			pendingVersionArchive, err := version.GetAppVersionArchive(appID, pendingVersion.ParentSequence)
			if err != nil {
				logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer os.RemoveAll(pendingVersionArchive)

			_, err = gitops.CreateGitOpsCommit(downstreamGitOps, appSlug, appName, int(pendingVersion.ParentSequence), pendingVersionArchive, downstreamName)
			if err != nil {
				logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}(a.ID, d.ClusterID, a.Slug, a.Name, d.Name)

	JSON(w, http.StatusNoContent, "")
}

func ResetGitOps(w http.ResponseWriter, r *http.Request) {
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

	if err := gitops.ResetGitOps(); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusNoContent, "")
}
