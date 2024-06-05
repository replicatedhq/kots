package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
)

type GetPreflightResultResponse struct {
	PreflightProgress string                         `json:"preflightProgress,omitempty"`
	PreflightResult   preflighttypes.PreflightResult `json:"preflightResult"`
}

type GetPreflightCommandRequest struct {
	Origin string `json:"origin"`
}

type GetPreflightCommandResponse struct {
	Command []string `json:"command"`
}

func (h *Handler) GetPreflightResult(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.ParseInt(mux.Vars(r)["sequence"], 10, 64)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	result, err := store.GetStore().GetPreflightResults(foundApp.ID, sequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	progress, err := store.GetStore().GetPreflightProgress(foundApp.ID, sequence)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get preflight progress"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := GetPreflightResultResponse{
		PreflightResult:   *result,
		PreflightProgress: progress,
	}
	JSON(w, 200, response)
}

func (h *Handler) GetLatestPreflightResultsForSequenceZero(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app from slug"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	result, err := store.GetStore().GetPreflightResults(foundApp.ID, 0)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get preflight result"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	progress, err := store.GetStore().GetPreflightProgress(foundApp.ID, 0)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get preflight progress"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := GetPreflightResultResponse{
		PreflightResult:   *result,
		PreflightProgress: progress,
	}
	JSON(w, http.StatusOK, response)
}

func (h *Handler) IgnorePreflightRBACErrors(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse sequence number"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app from slug"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	status, err := store.GetStore().GetDownstreamVersionStatus(foundApp.ID, int64(sequence))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get downstream version status"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if status == storetypes.VersionPendingDownload {
		logger.Error(errors.Errorf("not ignoring preflight rbac errors for version %d because it's %s", sequence, status))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := store.GetStore().SetIgnorePreflightPermissionErrors(foundApp.ID, int64(sequence)); err != nil {
		logger.Error(errors.Wrap(err, "failed to ignore preflight permission errors"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create temp dir"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	removeArchiveDir := true
	defer func() {
		if removeArchiveDir {
			os.RemoveAll(archiveDir)
		}
	}()

	err = store.GetStore().GetAppVersionArchive(foundApp.ID, int64(sequence), archiveDir)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app version archive"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	removeArchiveDir = false
	go func() {
		defer os.RemoveAll(archiveDir)
		if err := preflight.Run(foundApp.ID, foundApp.Slug, int64(sequence), foundApp.IsAirgap, false, archiveDir); err != nil {
			logger.Error(errors.Wrap(err, "failed to run preflights"))
			return
		}
	}()

	JSON(w, http.StatusOK, struct{}{})
}

func (h *Handler) StartPreflightChecks(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse sequence number"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app from slug"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	status, err := store.GetStore().GetDownstreamVersionStatus(foundApp.ID, int64(sequence))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get downstream version status"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if status == storetypes.VersionPendingDownload {
		logger.Error(errors.Errorf("not running preflights for version %d because it's %s", sequence, status))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := store.GetStore().ResetPreflightResults(foundApp.ID, int64(sequence)); err != nil {
		logger.Error(errors.Wrap(err, "failed to reset preflight results"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	archiveDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create temp dir"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	removeArchiveDir := true
	defer func() {
		if removeArchiveDir {
			os.RemoveAll(archiveDir)
		}
	}()

	err = store.GetStore().GetAppVersionArchive(foundApp.ID, int64(sequence), archiveDir)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app version archive"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	removeArchiveDir = false
	go func() {
		defer os.RemoveAll(archiveDir)
		if err := preflight.Run(foundApp.ID, foundApp.Slug, int64(sequence), foundApp.IsAirgap, false, archiveDir); err != nil {
			logger.Error(errors.Wrap(err, "failed to run preflights"))
			return
		}
	}()

	JSON(w, http.StatusOK, struct{}{})
}

func (h *Handler) GetPreflightCommand(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.ParseInt(mux.Vars(r)["sequence"], 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse sequence"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	inCluster := r.URL.Query().Get("inCluster") == "true"

	getPreflightCommandRequest := GetPreflightCommandRequest{}
	if err := json.NewDecoder(r.Body).Decode(&getPreflightCommandRequest); err != nil {
		logger.Error(errors.Wrap(err, "failed to decode request body"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	archivePath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create temp dir"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(archivePath)

	err = store.GetStore().GetAppVersionArchive(foundApp.ID, sequence, archivePath)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app archive"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(archivePath)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to load kots kinds"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = preflight.CreateRenderedSpec(foundApp, sequence, getPreflightCommandRequest.Origin, inCluster, kotsKinds, archivePath)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to render preflight spec"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := GetPreflightCommandResponse{
		Command: preflight.GetPreflightCommand(foundApp.Slug),
	}

	JSON(w, http.StatusOK, response)
}

// PostPreflightStatus route is UNAUTHENTICATED
// This request comes from the `kubectl preflight` command.
func (h *Handler) PostPreflightStatus(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.ParseInt(mux.Vars(r)["sequence"], 10, 64)
	if err != nil {
		err = errors.Wrap(err, "failed to parse sequence")
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		err = errors.Wrap(err, "failed to get app from slug")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = errors.Wrap(err, "failed to read request body")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := store.GetStore().SetPreflightResults(foundApp.ID, sequence, b); err != nil {
		err = errors.Wrap(err, "failed to set preflight results")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(204)
}

func (h *Handler) PreflightsReports(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Debugf("failed to get app from slug: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	license, err := store.GetStore().GetLatestLicenseForApp(foundApp.ID)
	if err != nil {
		logger.Debugf("failed to get latest license for app: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(foundApp.ID)
	if err != nil {
		logger.Debugf("failed to get downstreams for app: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if len(downstreams) == 0 {
		err = errors.New("no downstreams for app")
		logger.Debugf("failed to get downstreams for app: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clusterID := downstreams[0].ClusterID

	go func() {
		if err := reporting.GetReporter().SubmitPreflightData(license, foundApp.ID, clusterID, 0, true, "", false, "", ""); err != nil {
			logger.Debugf("failed to submit preflight data: %v", err)
			return
		}
	}()

	JSON(w, http.StatusOK, struct{}{})
}
