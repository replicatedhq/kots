package handlers

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
)

type GetAppContentsResponse struct {
	Files map[string][]byte `json:"files"`
}

func (h *Handler) GetAppContents(w http.ResponseWriter, r *http.Request) {
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
		logger.Error(errors.Errorf("not returning contents for version %d because it's %s", sequence, status))
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

	// walk the parth, adding all to the files map
	// base64 decode these
	archiveFiles := map[string][]byte{}

	err = filepath.Walk(archivePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		archiveFiles[strings.TrimPrefix(path, archivePath)] = contents
		return nil
	})
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to walk archive"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	getAppContentsResponse := GetAppContentsResponse{
		Files: archiveFiles,
	}

	JSON(w, http.StatusOK, getAppContentsResponse)
}
