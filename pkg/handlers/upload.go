package handlers

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/version"
)

type UploadExistingAppRequest struct {
	Slug           string `json:"slug"`
	VersionLabel   string `json:"versionLabel,omitempty"`
	UpdateCursor   string `json:"updateCursor,omitempty"`
	Deploy         bool   `json:"deploy"`
	SkipPreflights bool   `json:"skipPreflights"`
}

type UploadResponse struct {
	Slug string `json:"slug"`
}

// UploadExistingApp can be used to upload a multipart form file to the existing app
// This is used in the KOTS CLI when calling kots upload ...
// NOTE: this uses special kots token authorization
func (h *Handler) UploadExistingApp(w http.ResponseWriter, r *http.Request) {
	if err := requireValidKOTSToken(w, r); err != nil {
		logger.Error(errors.Wrap(err, "failed to get valid token"))
		return
	}

	metadata := r.FormValue("metadata")
	uploadExistingAppRequest := UploadExistingAppRequest{}
	if err := json.NewDecoder(strings.NewReader(metadata)).Decode(&uploadExistingAppRequest); err != nil {
		logger.Error(errors.Wrap(err, "failed to decode request"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	archive, _, err := r.FormFile("file")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to read file from request"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tmpFile, err := ioutil.TempFile("", "kotsadm")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create temp file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(tmpFile, archive)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to copy file from request to temp file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpFile.Name())

	archiveDir, err := version.ExtractArchiveToTempDirectory(tmpFile.Name())
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to extract file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(archiveDir)

	// encrypt any plain text values
	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to load kotskinds"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if kotsKinds.ConfigValues != nil {
		if err := kotsKinds.EncryptConfigValues(); err != nil {
			logger.Error(errors.Wrap(err, "failed to encrypt config values"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		updated, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to marshal config values"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"), []byte(updated), 0644); err != nil {
			logger.Error(errors.Wrap(err, "failed to write config values"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	a, err := store.GetStore().GetAppFromSlug(uploadExistingAppRequest.Slug)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get app for slug %q", uploadExistingAppRequest.Slug))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get registry settings"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	app, err := store.GetStore().GetApp(a.ID)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get app %q", a.ID))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list downstreams"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(downstreams) == 0 {
		logger.Errorf("no downstreams found for deploying %s", a.Slug)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = render.RenderDir(archiveDir, app, downstreams, registrySettings, true)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to render app version"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newSequence, err := store.GetStore().CreateAppVersion(a.ID, &a.CurrentSequence, archiveDir, "KOTS Upload", false, &version.DownstreamGitOps{})
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create app version"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !uploadExistingAppRequest.SkipPreflights {
		if err := preflight.Run(a.ID, a.Slug, newSequence, a.IsAirgap, archiveDir); err != nil {
			logger.Error(errors.Wrap(err, "failed to get run preflights"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if uploadExistingAppRequest.Deploy {
		status, err := store.GetStore().GetStatusForVersion(a.ID, downstreams[0].ClusterID, newSequence)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get update downstream status"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if status == storetypes.VersionPendingConfig {
			logger.Error(errors.Errorf("not deploying version %d because it's %s", newSequence, status))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := version.DeployVersion(a.ID, newSequence); err != nil {
			logger.Error(errors.Wrap(err, "failed to deploy latest version"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	uploadResponse := UploadResponse{
		Slug: a.Slug,
	}

	JSON(w, http.StatusOK, uploadResponse)
}
