package handlers

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
)

type GetPreflightResultResponse struct {
	PreflightResult downstreamtypes.PreflightResult `json:"preflightResult"`
}

func GetPreflightResult(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.ParseInt(mux.Vars(r)["sequence"], 10, 64)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	foundApp, err := app.GetFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	result, err := downstream.GetPreflightResult(foundApp.ID, sequence)
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
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	result, err := downstream.GetLatestPreflightResult()
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
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	foundApp, err := app.GetFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := downstream.SetIgnorePreflightPermissionErrors(foundApp.ID, int64(sequence)); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	archiveDir, err := version.GetAppVersionArchive(foundApp.ID, int64(sequence))
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
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	foundApp, err := app.GetFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := preflight.ResetPreflightResult(foundApp.ID, int64(sequence)); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	archiveDir, err := version.GetAppVersionArchive(foundApp.ID, int64(sequence))
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
