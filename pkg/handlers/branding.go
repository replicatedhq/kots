package handlers

import (
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
)

func (h *Handler) UploadInitialBranding(w http.ResponseWriter, r *http.Request) {
	if err := requireValidKOTSToken(w, r); err != nil {
		logger.Error(errors.Wrap(err, "failed to validate token"))
		return
	}

	archiveFile, _, err := r.FormFile("brandingArchive")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get form file reader"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer archiveFile.Close()

	brandingArchive, err := io.ReadAll(archiveFile)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to read form file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = store.GetStore().CreateInitialBranding(brandingArchive)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create initial branding"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
