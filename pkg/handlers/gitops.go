package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/gitops"
	"github.com/replicatedhq/kots/pkg/handlers/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/tasks"
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
	HTTPPort string `json:"httpPort"`
	SSHPort  string `json:"sshPort"`
}

func (h *Handler) UpdateAppGitOps(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) DisableAppGitOps(w http.ResponseWriter, r *http.Request) {
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
		err := gitops.DisableDownstreamGitOps(a.ID, clusterID, downstreamGitOps)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	go func() {
		err := reporting.GetReporter().SubmitAppInfo(appID)
		if err != nil {
			logger.Debugf("failed to submit app info: %v", err)
		}
	}()

	JSON(w, http.StatusNoContent, "")
}

func (h *Handler) InitGitOpsConnection(w http.ResponseWriter, r *http.Request) {
	currentStatus, _, err := tasks.GetTaskStatus("gitops-init")
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

	defaultBranchName, err := gitops.TestGitOpsConnection(downstreamGitOps)
	if err != nil {
		logger.Infof("Failed to test gitops connection: %v", err)

		if err := gitops.SetGitOpsError(a.ID, d.ClusterID, err.Error()); err != nil {
			logger.Error(err)
		}

		JSON(w, http.StatusBadRequest, types.NewErrorResponse(err))
		return
	}

	// If a branch is not provided, use the default branch
	if downstreamGitOps.Branch == "" {
		err := gitops.UpdateDownstreamGitOps(a.ID, d.ClusterID, downstreamGitOps.RepoURI, defaultBranchName,
			downstreamGitOps.Path, downstreamGitOps.Format, downstreamGitOps.Action)
		if err != nil {
			logger.Infof("Failed to update the gitops configmap with the default branch: %v", err)

			if err := gitops.SetGitOpsError(a.ID, d.ClusterID, err.Error()); err != nil {
				logger.Error(err)
			}
			JSON(w, http.StatusInternalServerError, types.NewErrorResponse(err))
			return
		}
		downstreamGitOps.Branch = defaultBranchName
	}

	if err := gitops.SetGitOpsError(a.ID, d.ClusterID, ""); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go func() {
		if err := tasks.SetTaskStatus("gitops-init", "Creating commits ...", "running"); err != nil {
			logger.Error(errors.Wrap(err, "failed to set task status running"))
			return
		}

		var finalError error
		defer func() {
			if finalError == nil {
				if err := tasks.ClearTaskStatus("gitops-init"); err != nil {
					logger.Error(errors.Wrap(err, "failed to clear task status"))
				}
			} else {
				if err := tasks.SetTaskStatus("gitops-init", finalError.Error(), "failed"); err != nil {
					logger.Error(errors.Wrap(err, "failed to set task status error"))
				}
			}
		}()

		appVersions, err := store.GetStore().GetDownstreamVersions(a.ID, d.ClusterID, true)
		if err != nil {
			err = errors.Wrap(err, "failed to get downstream versions")
			logger.Error(err)
			finalError = err
			return
		}

		// Create git commit for current version (if exists)
		if appVersions.CurrentVersion != nil {
			currentVersionArchive, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				err = errors.Wrap(err, "failed to create temp dir")
				logger.Error(err)
				finalError = err
				return
			}
			defer os.RemoveAll(currentVersionArchive)

			err = store.GetStore().GetAppVersionArchive(a.ID, appVersions.CurrentVersion.ParentSequence, currentVersionArchive)
			if err != nil {
				err = errors.Wrapf(err, "failed to get app version archive for current version %d", appVersions.CurrentVersion.ParentSequence)
				logger.Error(err)
				finalError = err
				return
			}

			_, err = gitops.CreateGitOpsCommit(downstreamGitOps, a.Slug, a.Name, int(appVersions.CurrentVersion.ParentSequence), currentVersionArchive, d.Name)
			if err != nil {
				err = errors.Wrapf(err, "failed to create gitops commit for current version %d", appVersions.CurrentVersion.ParentSequence)
				logger.Error(err)
				finalError = err
				return
			}
		}

		// Create git commits for sorted pending versions
		for _, pendingVersion := range appVersions.PendingVersions {
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

	go func() {
		err := reporting.GetReporter().SubmitAppInfo(appID)
		if err != nil {
			logger.Debugf("failed to submit app info: %v", err)
		}
	}()

	JSON(w, http.StatusNoContent, "")
}

func (h *Handler) ResetGitOps(w http.ResponseWriter, r *http.Request) {
	if err := gitops.ResetGitOps(); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusNoContent, "")
}

func (h *Handler) GetGitOpsRepo(w http.ResponseWriter, r *http.Request) {
	gitOpsConfig, err := gitops.GetGitOps()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, gitOpsConfig)
}

func (h *Handler) CreateGitOps(w http.ResponseWriter, r *http.Request) {
	createGitOpsRequest := CreateGitOpsRequest{}
	if err := json.NewDecoder(r.Body).Decode(&createGitOpsRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gitOpsInput := createGitOpsRequest.GitOpsInput
	if err := gitops.CreateGitOps(gitOpsInput.Provider, gitOpsInput.URI, gitOpsInput.Hostname, gitOpsInput.HTTPPort, gitOpsInput.SSHPort); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusNoContent, "")
}
