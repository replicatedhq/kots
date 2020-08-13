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
	"github.com/replicatedhq/kots/kotsadm/pkg/task"
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

	currentStatus, _, err := task.GetTaskStatus("gitops-init")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if currentStatus == "running" {
		logger.Error(errors.New("gitops-init is already running, not starting a new one"))
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

	go func() {
		if err := task.SetTaskStatus("gitops-init", "Creating commits ...", "running"); err != nil {
			logger.Error(errors.Wrap(err, "failed to set task status running"))
			return
		}

		var finalError error
		defer func() {
			if finalError == nil {
				if err := task.ClearTaskStatus("gitops-init"); err != nil {
					logger.Error(errors.Wrap(err, "failed to clear task status"))
				}
			} else {
				if err := task.SetTaskStatus("gitops-init", finalError.Error(), "failed"); err != nil {
					logger.Error(errors.Wrap(err, "failed to set task status error"))
				}
			}
		}()

		currentVersion, err := downstream.GetCurrentVersion(a.ID, d.ClusterID)
		if err != nil {
			err = errors.Wrap(err, "failed to get downstream current version")
			logger.Error(err)
			finalError = err
			return
		}

		pendingVersions, err := downstream.GetPendingVersions(a.ID, d.ClusterID)
		if err != nil {
			err = errors.Wrap(err, "failed to get downstream pending versions")
			logger.Error(err)
			finalError = err
			return
		}

		// Create git commit for current version
		currentVersionArchive, err := version.GetAppVersionArchive(a.ID, currentVersion.ParentSequence)
		if err != nil {
			err = errors.Wrapf(err, "failed to get app version archive for current version %d", currentVersion.ParentSequence)
			logger.Error(err)
			finalError = err
			return
		}
		defer os.RemoveAll(currentVersionArchive)

		_, err = gitops.CreateGitOpsCommit(downstreamGitOps, a.Slug, a.Name, int(currentVersion.ParentSequence), currentVersionArchive, d.Name)
		if err != nil {
			err = errors.Wrapf(err, "failed to create gitops commit for current version %d", currentVersion.ParentSequence)
			logger.Error(err)
			finalError = err
			return
		}

		// Sort pending versions ascending before creating commits
		sort.Slice(pendingVersions, func(i, j int) bool {
			return pendingVersions[i].ParentSequence < pendingVersions[j].ParentSequence
		})
		// Create git commits for sorted pending versions
		for _, pendingVersion := range pendingVersions {
			pendingVersionArchive, err := version.GetAppVersionArchive(a.ID, pendingVersion.ParentSequence)
			if err != nil {
				err = errors.Wrapf(err, "failed to get app version archive for pending version %d", pendingVersion.ParentSequence)
				logger.Error(err)
				finalError = err
				return
			}
			defer os.RemoveAll(pendingVersionArchive)

			_, err = gitops.CreateGitOpsCommit(downstreamGitOps, a.Slug, a.Name, int(pendingVersion.ParentSequence), pendingVersionArchive, d.Name)
			if err != nil {
				err = errors.Wrapf(err, "failed to create gitops commit for pending version %d", pendingVersion.ParentSequence)
				logger.Error(err)
				finalError = err
				return
			}
		}
	}()

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
