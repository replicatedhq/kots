package handlers

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/version"
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

	archiveFiles, err := version.GetAppVersionArchiveFiles(appSlug, int64(sequence))
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get archive files for app %s sequence %d", appSlug, sequence))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	getAppContentsResponse := GetAppContentsResponse{
		Files: archiveFiles,
	}

	JSON(w, http.StatusOK, getAppContentsResponse)
}
