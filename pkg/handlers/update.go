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
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	"github.com/replicatedhq/kots/pkg/util"
)

type AppUpdateCheckRequest struct {
}

type AppUpdateCheckResponse struct {
	AvailableUpdates   int64              `json:"availableUpdates"`
	CurrentAppSequence int64              `json:"currentAppSequence"`
	CurrentRelease     *AppUpdateRelease  `json:"currentRelease,omitempty"`
	AvailableReleases  []AppUpdateRelease `json:"availableReleases"`
	DeployingRelease   *AppUpdateRelease  `json:"deployingRelease,omitempty"`
}

type AppUpdateRelease struct {
	Sequence int64  `json:"sequence"`
	Version  string `json:"version"`
}

func (h *Handler) AppUpdateCheck(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get app for slug %q", appSlug))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	deploy, _ := strconv.ParseBool(r.URL.Query().Get("deploy"))
	deployVersionLabel := r.URL.Query().Get("deployVersionLabel")
	skipPreflights, _ := strconv.ParseBool(r.URL.Query().Get("skipPreflights"))
	skipCompatibilityCheck, _ := strconv.ParseBool(r.URL.Query().Get("skipCompatibilityCheck"))
	isCLI, _ := strconv.ParseBool(r.URL.Query().Get("isCLI"))
	wait, _ := strconv.ParseBool(r.URL.Query().Get("wait"))

	contentType := strings.Split(r.Header.Get("Content-Type"), ";")[0]
	contentType = strings.TrimSpace(contentType)

	if contentType == "application/json" {
		opts := updatechecker.CheckForUpdatesOpts{
			AppID:                  foundApp.ID,
			DeployLatest:           deploy,
			DeployVersionLabel:     deployVersionLabel,
			SkipPreflights:         skipPreflights,
			SkipCompatibilityCheck: skipCompatibilityCheck,
			IsCLI:                  isCLI,
			Wait:                   wait,
		}
		ucr, err := updatechecker.CheckForUpdates(opts)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to check for updates"))
			w.WriteHeader(http.StatusInternalServerError)

			cause := errors.Cause(err)
			if _, ok := cause.(util.ActionableError); ok {
				w.Write([]byte(cause.Error()))
			}
			return
		}

		// refresh the app to get the correct sequence
		a, err := store.GetStore().GetApp(foundApp.ID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get app"))
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var appUpdateCheckResponse AppUpdateCheckResponse
		if ucr != nil {
			var availableReleases []AppUpdateRelease
			for _, r := range ucr.AvailableReleases {
				availableReleases = append(availableReleases, AppUpdateRelease{
					Sequence: r.Sequence,
					Version:  r.Version,
				})
			}

			appUpdateCheckResponse = AppUpdateCheckResponse{
				AvailableUpdates:   ucr.AvailableUpdates,
				CurrentAppSequence: a.CurrentSequence,
				AvailableReleases:  availableReleases,
			}

			if ucr.CurrentRelease != nil {
				appUpdateCheckResponse.CurrentRelease = &AppUpdateRelease{
					Sequence: ucr.CurrentRelease.Sequence,
					Version:  ucr.CurrentRelease.Version,
				}
			}
			if ucr.DeployingRelease != nil {
				appUpdateCheckResponse.DeployingRelease = &AppUpdateRelease{
					Sequence: ucr.DeployingRelease.Sequence,
					Version:  ucr.DeployingRelease.Version,
				}
			}
		}

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

		err = airgap.UpdateAppFromPath(foundApp, rootDir, "", deploy, skipPreflights, skipCompatibilityCheck)
		if err != nil {
			finishedChan <- err

			logger.Error(errors.Wrap(err, "failed to upgrade app"))
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
