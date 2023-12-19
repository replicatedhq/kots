package handlers

import (
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/apparchive"
	"github.com/replicatedhq/kots/pkg/kotsutil"
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

	_, appFilesMap, err := apparchive.GetRenderedApp(archivePath, d.Name, kotsKinds.GetKustomizeBinaryPath())
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get rendered app"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, v1Beta1ChartsFilesMap, err := apparchive.GetRenderedV1Beta1ChartsArchive(archivePath, d.Name, kotsKinds.GetKustomizeBinaryPath())
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get rendered v1beta1 chart files"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	v1Beta2ChartsFilesMap, err := apparchive.GetRenderedV1Beta2FileMap(archivePath, d.Name)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get rendered v1beta2 chart files"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responseFiles := map[string]string{}
	for filename, content := range appFilesMap {
		responseFiles[filename] = string(content)
	}
	for filename, content := range v1Beta1ChartsFilesMap {
		responseFiles[filename] = string(content)
	}
	for filename, content := range v1Beta2ChartsFilesMap {
		responseFiles[filename] = string(content)
	}

	JSON(w, http.StatusOK, GetAppRenderedContentsResponse{
		Files: responseFiles,
	})
}
