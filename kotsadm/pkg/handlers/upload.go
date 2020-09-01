package handlers

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
)

type UploadExistingAppRequest struct {
	Slug         string `json:"slug"`
	VersionLabel string `json:"versionLabel,omitempty"`
	UpdateCursor string `json:"updateCursor,omitempty"`
}

type UploadResponse struct {
	Slug string `json:"slug"`
}

// UploadExistingApp can be used to upload a multipart form file to the existing app
// This is used in the KOTS CLI when calling kots upload ...
// NOTE: this uses special kots token authorization
func UploadExistingApp(w http.ResponseWriter, r *http.Request) {
	if err := requireValidKOTSToken(w, r); err != nil {
		logger.Error(err)
		return
	}

	metadata := r.FormValue("metadata")
	uploadExistingAppRequest := UploadExistingAppRequest{}
	if err := json.NewDecoder(strings.NewReader(metadata)).Decode(&uploadExistingAppRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	archive, _, err := r.FormFile("file")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	tmpFile, err := ioutil.TempFile("", "kotsadm")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	_, err = io.Copy(tmpFile, archive)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	defer os.RemoveAll(tmpFile.Name())

	archiveDir, err := version.ExtractArchiveToTempDirectory(tmpFile.Name())
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	defer os.RemoveAll(archiveDir)

	// encrypt any plain text values
	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := kotsKinds.EncryptConfigValues(); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	updated, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"), []byte(updated), 0644); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	a, err := store.GetStore().GetAppFromSlug(uploadExistingAppRequest.Slug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	app, err := store.GetStore().GetApp(a.ID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	err = render.RenderDir(archiveDir, app, downstreams, registrySettings)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	newSequence, err := version.CreateVersion(a.ID, archiveDir, "KOTS Upload", a.CurrentSequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := preflight.Run(a.ID, newSequence, a.IsAirgap, archiveDir); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	uploadResponse := UploadResponse{
		Slug: a.Slug,
	}

	JSON(w, 200, uploadResponse)
}
