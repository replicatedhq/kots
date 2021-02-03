package handlers

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/airgap"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	"github.com/replicatedhq/kots/pkg/util"
)

type AppUpdateCheckRequest struct {
}

type AppUpdateCheckResponse struct {
	AvailableUpdates   int64 `json:"availableUpdates"`
	CurrentAppSequence int64 `json:"currentAppSequence"`
}

func (h *Handler) AppUpdateCheck(w http.ResponseWriter, r *http.Request) {
	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	deploy, _ := strconv.ParseBool(r.URL.Query().Get("deploy"))
	skipPreflights, _ := strconv.ParseBool(r.URL.Query().Get("skipPreflights"))

	contentType := strings.Split(r.Header.Get("Content-Type"), ";")[0]
	contentType = strings.TrimSpace(contentType)

	if contentType == "application/json" {
		availableUpdates, err := updatechecker.CheckForUpdates(foundApp.ID, deploy, skipPreflights)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)

			cause := errors.Cause(err)
			if _, ok := cause.(util.ActionableError); ok {
				w.Write([]byte(cause.Error()))
			}
			return
		}

		appUpdateCheckResponse := AppUpdateCheckResponse{
			AvailableUpdates:   availableUpdates,
			CurrentAppSequence: foundApp.CurrentSequence,
		}

		// preflights reporting
		go func() {
			isUpdate := true
			err = reporting.SendPreflightInfo(foundApp.ID, int(foundApp.CurrentSequence), skipPreflights, isUpdate)
			if err != nil {
				logger.Debugf("failed to update preflights reports: %v", err)
			}
		}()

		JSON(w, http.StatusOK, appUpdateCheckResponse)

		return
	}

	if contentType == "multipart/form-data" {
		if !foundApp.IsAirgap {
			logger.Error(errors.New("not an airgap app"))
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Cannot update an online install using an airgap bundle"))
			return
		}

		rootDir, err := ioutil.TempDir("", "kotsadm-airgap")
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to create temp dir"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(rootDir)

		formReader, err := r.MultipartReader()
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get multipart reader"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for {
			part, err := formReader.NextPart()
			if err != nil {
				if err == io.EOF {
					break
				}
				logger.Error(errors.Wrap(err, "failed to get next part"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			fileName := filepath.Join(rootDir, part.FormName())
			file, err := os.Create(fileName)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to create file"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer file.Close()

			_, err = io.Copy(file, part)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to copy part data"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			file.Close()
		}

		finishedChan := make(chan error)
		defer close(finishedChan)

		airgap.StartUpdateTaskMonitor(finishedChan)

		err = airgap.UpdateAppFromPath(foundApp, rootDir, "", deploy, skipPreflights)
		if err != nil {
			finishedChan <- err

			logger.Error(errors.Wrap(err, "failed to upgrde app"))
			w.WriteHeader(http.StatusInternalServerError)

			cause := errors.Cause(err)
			if _, ok := cause.(util.ActionableError); ok {
				w.Write([]byte(cause.Error()))
			}
			return
		}

		JSON(w, http.StatusOK, struct{}{})

		return
	}

	logger.Error(errors.Errorf("unsupported content type: %s", r.Header.Get("Content-Type")))
	w.WriteHeader(http.StatusBadRequest)
}
