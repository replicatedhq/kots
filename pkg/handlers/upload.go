package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/handlers/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/render"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
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
	types.ErrorResponse
	Slug *string `json:"slug,omitempty"`
}

// UploadExistingApp can be used to upload a multipart form file to the existing app
// This is used in the KOTS CLI when calling kots upload ...
// NOTE: this uses special kots token authorization
func (h *Handler) UploadExistingApp(w http.ResponseWriter, r *http.Request) {
	uploadResponse := UploadResponse{}

	if err := requireValidKOTSToken(w, r); err != nil {
		logger.Error(errors.Wrap(err, "failed to get valid token"))
		return
	}

	metadata := r.FormValue("metadata")
	uploadExistingAppRequest := UploadExistingAppRequest{}
	if err := json.NewDecoder(strings.NewReader(metadata)).Decode(&uploadExistingAppRequest); err != nil {
		uploadResponse.Error = util.StrPointer("failed to decode request")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}

	archive, _, err := r.FormFile("file")
	if err != nil {
		uploadResponse.Error = util.StrPointer("failed to read file from request")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}

	tmpFile, err := os.CreateTemp("", "kotsadm")
	if err != nil {
		uploadResponse.Error = util.StrPointer("failed to create temp file")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}
	_, err = io.Copy(tmpFile, archive)
	if err != nil {
		uploadResponse.Error = util.StrPointer("failed to copy file from request to temp file")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}
	defer os.RemoveAll(tmpFile.Name())

	archiveDir, err := version.ExtractArchiveToTempDirectory(tmpFile.Name())
	if err != nil {
		uploadResponse.Error = util.StrPointer("failed to extract file")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}
	defer os.RemoveAll(archiveDir)

	// encrypt any plain text values
	kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
	if err != nil {
		uploadResponse.Error = util.StrPointer("failed to load kotskinds")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}

	if kotsKinds.ConfigValues != nil {
		if err := kotsKinds.EncryptConfigValues(); err != nil {
			uploadResponse.Error = util.StrPointer("failed to encrypt config values")
			logger.Error(errors.Wrap(err, *uploadResponse.Error))
			JSON(w, http.StatusInternalServerError, uploadResponse)
			return
		}
		updated, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
		if err != nil {
			uploadResponse.Error = util.StrPointer("failed to marshal config values")
			logger.Error(errors.Wrap(err, *uploadResponse.Error))
			JSON(w, http.StatusInternalServerError, uploadResponse)
			return
		}

		if err := os.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"), []byte(updated), 0644); err != nil {
			uploadResponse.Error = util.StrPointer("failed to write config values")
			logger.Error(errors.Wrap(err, *uploadResponse.Error))
			JSON(w, http.StatusInternalServerError, uploadResponse)
			return
		}
	}

	a, err := store.GetStore().GetAppFromSlug(uploadExistingAppRequest.Slug)
	if err != nil {
		uploadResponse.Error = util.StrPointer(fmt.Sprintf("failed to get app for slug %q", uploadExistingAppRequest.Slug))
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		uploadResponse.Error = util.StrPointer("failed to get registry settings")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		uploadResponse.Error = util.StrPointer("failed to list downstreams")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}

	if len(downstreams) == 0 {
		uploadResponse.Error = util.StrPointer(fmt.Sprintf("no downstreams found for deploying %s", a.Slug))
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}

	nextAppSequence, err := store.GetStore().GetNextAppSequence(a.ID)
	if err != nil {
		uploadResponse.Error = util.StrPointer("failed to get next app sequence")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}

	err = render.RenderDir(rendertypes.RenderDirOptions{
		ArchiveDir:       archiveDir,
		App:              a,
		Downstreams:      downstreams,
		RegistrySettings: registrySettings,
		Sequence:         nextAppSequence,
		ReportingInfo:    reporting.GetReportingInfo(a.ID),
	})
	if err != nil {
		cause := errors.Cause(err)
		if _, ok := cause.(util.ActionableError); ok {
			uploadResponse.Error = util.StrPointer(cause.Error())
			logger.Error(errors.Wrap(err, *uploadResponse.Error))
			JSON(w, http.StatusInternalServerError, uploadResponse)
		} else {
			uploadResponse.Error = util.StrPointer("failed to render app version")
			logger.Error(errors.Wrap(err, *uploadResponse.Error))
			JSON(w, http.StatusInternalServerError, uploadResponse)
		}
		return
	}

	baseSequence, err := store.GetStore().GetAppVersionBaseSequence(a.ID, kotsKinds.Installation.Spec.VersionLabel)
	if err != nil {
		uploadResponse.Error = util.StrPointer("failed to app version base sequence")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}

	newSequence, err := store.GetStore().CreateAppVersion(a.ID, &baseSequence, archiveDir, "KOTS Upload", false, false, "", uploadExistingAppRequest.SkipPreflights, render.Renderer{})
	if err != nil {
		uploadResponse.Error = util.StrPointer("failed to create app version")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}

	status, err := store.GetStore().GetDownstreamVersionStatus(a.ID, newSequence)
	if err != nil {
		uploadResponse.Error = util.StrPointer("failed to get downstream version status")
		logger.Error(errors.Wrap(err, *uploadResponse.Error))
		JSON(w, http.StatusInternalServerError, uploadResponse)
		return
	}
	if status == storetypes.VersionPendingPreflight {
		if err := preflight.Run(a.ID, a.Slug, newSequence, a.IsAirgap, uploadExistingAppRequest.SkipPreflights, archiveDir); err != nil {
			uploadResponse.Error = util.StrPointer("failed to get run preflights")
			logger.Error(errors.Wrap(err, *uploadResponse.Error))
			JSON(w, http.StatusInternalServerError, uploadResponse)
			return
		}
	}

	if uploadExistingAppRequest.Deploy {
		if err := version.DeployVersion(a.ID, newSequence); err != nil {
			cause := errors.Cause(err)
			if _, ok := cause.(util.ActionableError); ok {
				uploadResponse.Error = util.StrPointer(cause.Error())
				logger.Error(errors.Wrap(err, *uploadResponse.Error))
				JSON(w, http.StatusInternalServerError, uploadResponse)
			} else {
				uploadResponse.Error = util.StrPointer("failed to deploy latest version")
				logger.Error(errors.Wrap(err, *uploadResponse.Error))
				JSON(w, http.StatusInternalServerError, uploadResponse)
			}
			return
		}
	}

	uploadResponse.Slug = util.StrPointer(a.Slug)
	uploadResponse.Success = true
	uploadResponse.Error = nil

	JSON(w, http.StatusOK, uploadResponse)
}
