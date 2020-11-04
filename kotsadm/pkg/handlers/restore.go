package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshot"
	snapshottypes "github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
)

type CreateApplicationRestoreResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type GetRestoreStatusResponse struct {
	Status      string `json:"status,omitempty"`
	RestoreName string `json:"restore_name,omitempty"`
	Error       string `json:"error,omitempty"`
}

func CreateApplicationRestore(w http.ResponseWriter, r *http.Request) {
	createRestoreResponse := CreateApplicationRestoreResponse{
		Success: false,
	}

	snapshotName := mux.Vars(r)["snapshotName"]

	backup, err := snapshot.GetBackup(snapshotName)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to find backup"
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	appID := backup.Annotations["kots.io/app-id"]
	sequence, err := strconv.ParseInt(backup.Annotations["kots.io/app-sequence"], 10, 64)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to parse sequence label"
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	status, err := downstream.GetDownstreamVersionStatus(appID, sequence)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to find downstream version"
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	if status != "deployed" {
		err := errors.Errorf("sequence %d of app %s was never deployed to this cluster", sequence, appID)
		logger.Error(err)
		createRestoreResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	kotsApp, err := store.GetStore().GetApp(appID)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to get app"
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	if kotsApp.RestoreInProgressName != "" {
		err := errors.Errorf("restore is already in progress")
		logger.Error(err)
		createRestoreResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	if err := snapshot.DeleteRestore(snapshotName); err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to delete restore"
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	err = app.InitiateRestore(snapshotName, appID)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to initiate restore"
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	createRestoreResponse.Success = true

	JSON(w, http.StatusOK, createRestoreResponse)
}

type RestoreAppsResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func RestoreApps(w http.ResponseWriter, r *http.Request) {
	restoreResponse := RestoreAppsResponse{
		Success: false,
	}

	snapshotName := mux.Vars(r)["snapshotName"]

	backup, err := snapshot.GetBackup(snapshotName)
	if err != nil {
		logger.Error(err)
		restoreResponse.Error = "failed to find backup"
		JSON(w, http.StatusInternalServerError, restoreResponse)
		return
	}

	if backup.Annotations["kots.io/instance"] != "true" {
		err := errors.Errorf("backup %s is not an instance backup", backup.ObjectMeta.Name)
		logger.Error(err)
		restoreResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, restoreResponse)
		return
	}

	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		logger.Error(err)
		restoreResponse.Error = "failed to list installed apps"
		JSON(w, http.StatusInternalServerError, restoreResponse)
		return
	}

	for _, a := range apps {
		if err := app.ResetRestore(a.ID); err != nil {
			logger.Error(err)
			restoreResponse.Error = fmt.Sprintf("failed to reset restore for app %s", a.Slug)
			JSON(w, http.StatusInternalServerError, restoreResponse)
			return
		}

		restoreName := fmt.Sprintf("%s.%s", snapshotName, a.Slug)
		if err := snapshot.DeleteRestore(restoreName); err != nil {
			logger.Error(err)
			restoreResponse.Error = fmt.Sprintf("failed to delete restore for app %s", a.Slug)
			JSON(w, http.StatusInternalServerError, restoreResponse)
			return
		}

		if err := app.InitiateRestore(snapshotName, a.ID); err != nil {
			logger.Error(err)
			restoreResponse.Error = fmt.Sprintf("failed to initiate restore for app %s", a.Slug)
			JSON(w, http.StatusInternalServerError, restoreResponse)
			return
		}
	}

	restoreResponse.Success = true

	JSON(w, http.StatusOK, restoreResponse)
}

type GetRestoreAppsStatusResponse struct {
	Statuses []AppRestoreStatus `json:"statuses"`
	Error    string             `json:"error,omitempty"`
}
type AppRestoreStatus struct {
	AppSlug string                 `json:"appSlug"`
	Status  velerov1.RestoreStatus `json:"status,omitempty"`
}

func GetRestoreAppsStatus(w http.ResponseWriter, r *http.Request) {
	response := GetRestoreAppsStatusResponse{
		Statuses: []AppRestoreStatus{},
	}

	snapshotName := mux.Vars(r)["snapshotName"]

	backup, err := snapshot.GetBackup(snapshotName)
	if err != nil {
		logger.Error(err)
		response.Error = "failed to find backup"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if backup.Annotations["kots.io/instance"] != "true" {
		err := errors.Errorf("backup %s is not an instance backup", backup.ObjectMeta.Name)
		logger.Error(err)
		response.Error = err.Error()
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		logger.Error(err)
		response.Error = "failed to list installed apps"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	statuses := []AppRestoreStatus{}

	for _, a := range apps {
		restoreName := fmt.Sprintf("%s.%s", snapshotName, a.Slug)
		restore, err := snapshot.GetRestore(restoreName)
		if err != nil {
			logger.Error(err)
			response.Error = fmt.Sprintf("failed to get restore for app %s", a.Slug)
			JSON(w, http.StatusInternalServerError, response)
			return
		}

		appRestoreStatus := AppRestoreStatus{
			AppSlug: a.Slug,
		}

		if restore == nil {
			statuses = append(statuses, appRestoreStatus)
			continue
		}

		appRestoreStatus.Status = restore.Status
		statuses = append(statuses, appRestoreStatus)
	}

	response.Statuses = statuses

	JSON(w, http.StatusOK, response)
}

func GetRestoreStatus(w http.ResponseWriter, r *http.Request) {
	response := GetRestoreStatusResponse{
		Status: "",
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		response.Error = "failed to get app from app slug"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if foundApp.RestoreInProgressName != "" {
		response.RestoreName = foundApp.RestoreInProgressName
		response.Status = "running" // there is only one status right now
	}

	JSON(w, http.StatusOK, response)
}

func CancelRestore(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		err = errors.Wrap(err, "failed to get app from app slug")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := app.ResetRestore(foundApp.ID); err != nil {
		err = errors.Wrap(err, "failed to reset app restore in progress name")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type GetRestoreDetailsResponse struct {
	RestoreDetail *snapshottypes.RestoreDetail `json:"restoreDetail"`
	IsActive      bool                         `json:"active"`
}

func GetRestoreDetails(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	restoreName := mux.Vars(r)["restoreName"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		err = errors.Wrap(err, "failed to get app from app slug")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := GetRestoreDetailsResponse{
		IsActive: foundApp.RestoreInProgressName == restoreName,
	}

	restoreDetail, err := snapshot.GetRestoreDetails(context.TODO(), restoreName)
	if kuberneteserrors.IsNotFound(errors.Cause(err)) {
		if foundApp.RestoreUndeployStatus == apptypes.UndeployFailed {
			// HACK: once the user has see the error, clear it out.
			// Otherwise there is no way to get back to snapshot list.
			if err := app.ResetRestore(foundApp.ID); err != nil {
				err = errors.Wrap(err, "failed to reset app restore in progress name")
				logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.RestoreDetail = restoreDetail

	JSON(w, http.StatusOK, response)
}
