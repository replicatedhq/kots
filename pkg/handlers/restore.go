package handlers

import (
	"net/http"
	"strconv"

	"github.com/replicatedhq/kotsadm/pkg/app"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/session"
	"github.com/replicatedhq/kotsadm/pkg/snapshot"
)

type CreateRestoreRequest struct {
}

type CreateRestoreResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func CreateRestore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	createRestoreResponse := CreateRestoreResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to parse authorization header"
		JSON(w, 401, createRestoreResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		createRestoreResponse.Error = "failed to parse authorization header"
		JSON(w, 401, createRestoreResponse)
		return
	}

	backup, err := snapshot.GetBackup(mux.Vars(r)["snapshotName"])
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to find backup"
		JSON(w, 500, createRestoreResponse)
		return
	}

	appID := backup.Annotations["kots.io/app-id"]
	sequence, err := strconv.ParseInt(backup.Annotations["kots.io/app-sequence"], 10, 64)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to parse sequence label"
		JSON(w, 500, createRestoreResponse)
		return
	}

	status, err := downstream.GetDownstreamVersionStatus(appID, sequence)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to find downstream version"
		JSON(w, 500, createRestoreResponse)
		return
	}

	if status != "deployed" {
		err := errors.Errorf("sequence %d of app %s was never deployed to this cluster", sequence, appID)
		logger.Error(err)
		createRestoreResponse.Error = err.Error()
		JSON(w, 500, createRestoreResponse)
		return
	}

	kotsApp, err := app.Get(appID)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to get app"
		JSON(w, 500, createRestoreResponse)
		return
	}

	if kotsApp.RestoreInProgressName != "" {
		err := errors.Errorf("restore is already in progress")
		logger.Error(err)
		createRestoreResponse.Error = err.Error()
		JSON(w, 500, createRestoreResponse)
		return
	}

	err = app.InitiateRestore(mux.Vars(r)["snapshotName"], appID)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to initiate restore"
		JSON(w, 500, createRestoreResponse)
		return
	}

	createRestoreResponse.Success = true

	JSON(w, 200, createRestoreResponse)
}
