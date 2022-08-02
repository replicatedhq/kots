package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/marccampbell/yaml-toolbox/pkg/splitter"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kustomize"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
)

type GetAppRenderedContentsResponse struct {
	Files map[string]string `json:"files"`
}

type GetAppRenderedContentsErrorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) GetAppRenderedContents(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse sequence number"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app from slug"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	status, err := store.GetStore().GetDownstreamVersionStatus(a.ID, int64(sequence))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get downstream version status"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if status == storetypes.VersionPendingDownload {
		logger.Error(errors.Errorf("not returning rendered contents for version %d because it's %s", sequence, status))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	archivePath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create temp dir"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(archivePath)

	err = store.GetStore().GetAppVersionArchive(a.ID, int64(sequence), archivePath)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app version archive"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archivePath)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to load kots kinds from path"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to list downstreams for app %q", a.Slug))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(downstreams) == 0 {
		logger.Error(errors.Errorf("no downstreams found for app %q", a.Slug))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	d := downstreams[0]

	kustomizeBinPath := kotsKinds.GetKustomizeBinaryPath()
	kustomizeBuildTarget := filepath.Join(archivePath, "overlays", "downstreams", d.Name)

	archiveOutput, err := exec.Command(kustomizeBinPath, "build", kustomizeBuildTarget).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
			logger.Error(err)

			JSON(w, http.StatusInternalServerError, GetAppRenderedContentsErrorResponse{
				Error: fmt.Sprintf("Failed to build release: %v", err),
			})
			return
		}
		logger.Error(errors.Wrap(err, "failed to exec command"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	archiveFiles, err := splitter.SplitYAML(archiveOutput)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to split yaml"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// base64 decode these
	decodedArchiveFiles := map[string]string{}
	for filename, b := range archiveFiles {
		decodedArchiveFiles[filename] = string(b)
	}

	_, kustomizedFiles, err := kustomize.RenderChartsArchive(archivePath, d.Name, kustomizeBinPath)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get kustomized files"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for filename, b := range kustomizedFiles {
		decodedArchiveFiles[filename] = string(b)
	}

	JSON(w, http.StatusOK, GetAppRenderedContentsResponse{
		Files: decodedArchiveFiles,
	})
}
