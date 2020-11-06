package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"sort"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/pkg/gitops"
)

type UpdateAppGitOpsRequest struct {
	GitOpsInput UpdateAppGitOpsInput `json:"gitOpsInput"`
}
type UpdateAppGitOpsInput struct {
	URI    string `json:"uri"`
	Branch string `json:"branch"`
	Path   string `json:"path"`
	Format string `json:"format"`
	Action string `json:"action"`
}

type CreateGitOpsRequest struct {
	GitOpsInput CreateGitOpsInput `json:"gitOpsInput"`
}
type CreateGitOpsInput struct {
	Provider string `json:"provider"`
	URI      string `json:"uri"`
	Hostname string `json:"hostname"`
}

func UpdateAppGitOps(w http.ResponseWriter, r *http.Request) {
	updateAppGitOpsRequest := UpdateAppGitOpsRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateAppGitOpsRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	appID := mux.Vars(r)["appId"]
	clusterID := mux.Vars(r)["clusterId"]

	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gitOpsInput := updateAppGitOpsRequest.GitOpsInput
	if err := gitops.UpdateDownstreamGitOps(a.ID, clusterID, gitOpsInput.URI, gitOpsInput.Branch, gitOpsInput.Path, gitOpsInput.Format, gitOpsInput.Action); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusNoContent, "")
}

func DisableAppGitOps(w http.ResponseWriter, r *http.Request) {
	appID := mux.Vars(r)["appId"]
	clusterID := mux.Vars(r)["clusterId"]

	a, err := store.GetStore().GetApp(appID)
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
	currentStatus, _, err := store.GetStore().GetTaskStatus("gitops-init")
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

	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	d, err := store.GetStore().GetDownstream(clusterID)
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
		logger.Infof("Failed to test gitops connection: %v", err)

		if err := gitops.SetGitOpsError(a.ID, d.ClusterID, err.Error()); err != nil {
			logger.Error(err)
		}

		JSON(w, http.StatusBadRequest, NewErrorResponse(err))
		return
	}

	if err := gitops.SetGitOpsError(a.ID, d.ClusterID, ""); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go func() {
		if err := store.GetStore().SetTaskStatus("gitops-init", "Creating commits ...", "running"); err != nil {
			logger.Error(errors.Wrap(err, "failed to set task status running"))
			return
		}

		var finalError error
		defer func() {
			if finalError == nil {
				if err := store.GetStore().ClearTaskStatus("gitops-init"); err != nil {
					logger.Error(errors.Wrap(err, "failed to clear task status"))
				}
			} else {
				if err := store.GetStore().SetTaskStatus("gitops-init", finalError.Error(), "failed"); err != nil {
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

		// Create git commit for current version (if exists)
		if currentVersion != nil {
			currentVersionArchive, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				err = errors.Wrap(err, "failed to create temp dir")
				logger.Error(err)
				finalError = err
				return
			}
			defer os.RemoveAll(currentVersionArchive)

			err = store.GetStore().GetAppVersionArchive(a.ID, currentVersion.ParentSequence, currentVersionArchive)
			if err != nil {
				err = errors.Wrapf(err, "failed to get app version archive for current version %d", currentVersion.ParentSequence)
				logger.Error(err)
				finalError = err
				return
			}

			_, err = gitops.CreateGitOpsCommit(downstreamGitOps, a.Slug, a.Name, int(currentVersion.ParentSequence), currentVersionArchive, d.Name)
			if err != nil {
				err = errors.Wrapf(err, "failed to create gitops commit for current version %d", currentVersion.ParentSequence)
				logger.Error(err)
				finalError = err
				return
			}
		}

		// Sort pending versions ascending before creating commits
		sort.Slice(pendingVersions, func(i, j int) bool {
			return pendingVersions[i].ParentSequence < pendingVersions[j].ParentSequence
		})
		// Create git commits for sorted pending versions
		for _, pendingVersion := range pendingVersions {
			pendingVersionArchive, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				err = errors.Wrap(err, "failed to create temp dir")
				logger.Error(err)
				finalError = err
				return
			}
			defer os.RemoveAll(pendingVersionArchive)

			err = store.GetStore().GetAppVersionArchive(a.ID, pendingVersion.ParentSequence, pendingVersionArchive)
			if err != nil {
				err = errors.Wrapf(err, "failed to get app version archive for pending version %d", pendingVersion.ParentSequence)
				logger.Error(err)
				finalError = err
				return
			}

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
	if err := gitops.ResetGitOps(); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusNoContent, "")
}

func GetGitOpsRepo(w http.ResponseWriter, r *http.Request) {
	gitOpsConfig, err := gitops.GetGitOps()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, gitOpsConfig)
}

func CreateGitOps(w http.ResponseWriter, r *http.Request) {
	createGitOpsRequest := CreateGitOpsRequest{}
	if err := json.NewDecoder(r.Body).Decode(&createGitOpsRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gitOpsInput := createGitOpsRequest.GitOpsInput
	if err := gitops.CreateGitOps(gitOpsInput.Provider, gitOpsInput.URI, gitOpsInput.Hostname); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusNoContent, "")
}
