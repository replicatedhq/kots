package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	preflighttypes "github.com/replicatedhq/kots/kotsadm/pkg/preflight/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

type GetPreflightResultResponse struct {
	PreflightResult preflighttypes.PreflightResult `json:"preflightResult"`
}

type GetPreflightCommandResponse struct {
	Command []string `json:"command"`
}

func GetPreflightResult(w http.ResponseWriter, r *http.Request) {
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

	response := GetPreflightResultResponse{
		PreflightResult: *result,
	}
	JSON(w, 200, response)
}

func GetLatestPreflightResult(w http.ResponseWriter, r *http.Request) {
	result, err := store.GetStore().GetLatestPreflightResults()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	response := GetPreflightResultResponse{
		PreflightResult: *result,
	}
	JSON(w, 200, response)
}

func IgnorePreflightRBACErrors(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
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

	if err := store.GetStore().SetIgnorePreflightPermissionErrors(foundApp.ID, int64(sequence)); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	archiveDir, err := store.GetStore().GetAppVersionArchive(foundApp.ID, int64(sequence))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	go func() {
		defer os.RemoveAll(archiveDir)
		if err := preflight.Run(foundApp.ID, int64(sequence), archiveDir); err != nil {
			logger.Error(err)
			return
		}
	}()

	JSON(w, 200, struct{}{})
}

func StartPreflightChecks(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
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

	if err := store.GetStore().ResetPreflightResults(foundApp.ID, int64(sequence)); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	archiveDir, err := store.GetStore().GetAppVersionArchive(foundApp.ID, int64(sequence))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	go func() {
		defer os.RemoveAll(archiveDir)
		if err := preflight.Run(foundApp.ID, int64(sequence), archiveDir); err != nil {
			logger.Error(err)
			return
		}
	}()

	JSON(w, 200, struct{}{})
}

func GetPreflightCommand(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.ParseInt(mux.Vars(r)["sequence"], 10, 64)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	command := []string{
		"curl https://krew.sh/preflight | bash",
		fmt.Sprintf("kubectl preflight API_ADDRESS/api/v1/preflight/app/%s/sequence/%d", appSlug, sequence),
	}

	response := GetPreflightCommandResponse{
		Command: command,
	}
	JSON(w, 200, response)
}

// GetPreflightStatus route is UNAUTHENTICATED
// This request comes from the `kubectl preflight` command.
func GetPreflightStatus(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.ParseInt(mux.Vars(r)["sequence"], 10, 64)
	if err != nil {
		err = errors.Wrap(err, "failed to parse sequence")
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	inCluster := r.URL.Query().Get("inCluster")

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		err = errors.Wrap(err, "failed to get app from slug")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	archiveDir, err := store.GetStore().GetAppVersionArchive(foundApp.ID, sequence)
	if err != nil {
		err = errors.Wrap(err, "failed to get app version archive")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// we need a few objects from the app to check for updates
	renderedKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		err = errors.Wrap(err, "failed to load kotskinds")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if renderedKotsKinds.Preflight == nil {
		w.WriteHeader(404)
		return
	}

	// render the preflight file
	// we need to convert to bytes first, so that we can reuse the renderfile function
	renderedMarshalledPreflights, err := renderedKotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Preflight")
	if err != nil {
		err = errors.Wrap(err, "failed to marshal rendered preflight")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to get registry settings for app")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	renderedPreflight, err := render.RenderFile(renderedKotsKinds, registrySettings, []byte(renderedMarshalledPreflights))
	if err != nil {
		err = errors.Wrap(err, "failed to render preflights")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	specJSON, err := kotsutil.LoadPreflightFromContents(renderedPreflight)
	if err != nil {
		err = errors.Wrap(err, "failed to load rendered preflight")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	var baseURL string
	if inCluster == "true" {
		baseURL = os.Getenv("API_ENDPOINT")
	} else {
		baseURL = os.Getenv("API_ADVERTISE_ENDPOINT")
	}
	specJSON.Spec.UploadResultsTo = fmt.Sprintf("%s/api/v1/preflight/app/%s/sequence/%d", baseURL, appSlug, sequence)

	YAML(w, 200, specJSON)
}

// PostPreflightStatus route is UNAUTHENTICATED
// This request comes from the `kubectl preflight` command.
func PostPreflightStatus(w http.ResponseWriter, r *http.Request) {
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
