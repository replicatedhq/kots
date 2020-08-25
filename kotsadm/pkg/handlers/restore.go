package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshot"
	snapshottypes "github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
)

type CreateRestoreResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type GetRestoreStatusResponse struct {
	Status      string `json:"status,omitempty"`
	RestoreName string `json:"restore_name,omitempty"`
	Error       string `json:"error,omitempty"`
}

type CreateKotsadmRestoreResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func CreateRestore(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
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

	snapshotName := mux.Vars(r)["snapshotName"]

	backup, err := snapshot.GetBackup(snapshotName)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to find backup"
		JSON(w, 500, createRestoreResponse)
		return
	}

	if backup.Annotations[types.VeleroKey] == types.VeleroLabelConsoleValue {
		// this is a kotsadm snapshot being restored
		opts := &types.RestoreJobOptions{
			BackupName: snapshotName,
		}
		if err := kotsadm.CreateRestoreJob(opts); err != nil {
			logger.Error(err)
			createRestoreResponse.Error = "failed to initiate restore"
			JSON(w, 500, createRestoreResponse)
			return
		}

		createRestoreResponse.Success = true
		JSON(w, 200, createRestoreResponse)
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

	kotsApp, err := store.GetStore().GetApp(appID)
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

	if err := snapshot.DeleteRestore(snapshotName); err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to initiate restore"
		JSON(w, 500, createRestoreResponse)
		return
	}

	err = app.InitiateRestore(snapshotName, appID)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to initiate restore"
		JSON(w, 500, createRestoreResponse)
		return
	}

	createRestoreResponse.Success = true

	JSON(w, 200, createRestoreResponse)
}

func GetRestoreStatus(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	response := GetRestoreStatusResponse{
		Status: "",
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		response.Error = "failed to parse authorization header"
		JSON(w, 401, response)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		response.Error = "failed to parse authorization header"
		JSON(w, 401, response)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		response.Error = "failed to get app from app slug"
		JSON(w, 500, response)
		return
	}

	if foundApp.RestoreInProgressName != "" {
		response.RestoreName = foundApp.RestoreInProgressName
		response.Status = "running" // there is only one status right now
	}

	JSON(w, 200, response)
}

func CancelRestore(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		err = errors.Wrap(err, "failed to get app from app slug")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := app.ResetRestore(foundApp.ID); err != nil {
		err = errors.Wrap(err, "failed to reset app restore in progress name")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(204)
}

type GetKotsadmRestoreResponse struct {
	RestoreDetail *snapshottypes.RestoreDetail `json:"restoreDetail"`
	IsActive      bool                         `json:"active"`
}

func GetKotsadmRestore(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]
	restoreName := mux.Vars(r)["restoreName"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		err = errors.Wrap(err, "failed to get app from app slug")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	response := GetKotsadmRestoreResponse{
		IsActive: foundApp.RestoreInProgressName == restoreName,
	}

	restoreDetail, err := snapshot.GetKotsadmRestoreDetail(context.TODO(), restoreName)
	if kuberneteserrors.IsNotFound(errors.Cause(err)) {
		if foundApp.RestoreUndeployStatus == apptypes.UndeployFailed {
			// HACK: once the user has see the error, clear it out.
			// Otherwise there is no way to get back to snapshot list.
			if err := app.ResetRestore(foundApp.ID); err != nil {
				err = errors.Wrap(err, "failed to reset app restore in progress name")
				logger.Error(err)
				w.WriteHeader(500)
				return
			}
			restoreDetail = &snapshottypes.RestoreDetail{
				Name:  restoreName,
				Phase: string(velerov1.RestorePhaseFailed),
				Errors: []snapshottypes.SnapshotError{{
					Title:   "Restore has failed",
					Message: "Please check logs for errors.",
				}},
			}
		} else {
			restoreDetail = &snapshottypes.RestoreDetail{
				Name:  restoreName,
				Phase: string(velerov1.RestorePhaseNew),
			}
		}
	} else if err != nil {
		err = errors.Wrap(err, "failed to get restore detail")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	response.RestoreDetail = restoreDetail

	JSON(w, 200, response)
}
