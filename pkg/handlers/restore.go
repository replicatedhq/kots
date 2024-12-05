package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/app"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
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

func (h *Handler) CreateApplicationRestore(w http.ResponseWriter, r *http.Request) {
	createRestoreResponse := CreateApplicationRestoreResponse{
		Success: false,
	}

	appSlug := mux.Vars(r)["appSlug"]
	snapshotName := mux.Vars(r)["snapshotName"]
	kotsadmNamespace := util.PodNamespace

	backup, err := snapshot.GetBackup(r.Context(), kotsadmNamespace, snapshotName)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to find backup"
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	if snapshot.IsInstanceBackup(*backup) && snapshot.GetInstanceBackupType(*backup) != snapshottypes.InstanceBackupTypeLegacy {
		err := errors.New("only legacy type instance backups are restorable")
		logger.Error(err)
		createRestoreResponse.Error = err.Error()
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

	kotsApp, err := store.GetStore().GetApp(appID)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to get app"
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	if kotsApp.Slug != appSlug {
		err := errors.New(fmt.Sprintf("snapshot %s does not belong to app %s", snapshotName, appSlug))
		logger.Error(err)
		createRestoreResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	status, err := store.GetStore().GetDownstreamVersionStatus(kotsApp.ID, sequence)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to find downstream version"
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	if status != "deployed" {
		err := errors.Errorf("sequence %d of app %s was never deployed to this cluster", sequence, kotsApp.ID)
		logger.Error(err)
		createRestoreResponse.Error = err.Error()
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

	if err := snapshot.DeleteRestore(r.Context(), kotsadmNamespace, snapshotName); err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to delete restore"
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	err = app.InitiateRestore(snapshotName, kotsApp.ID)
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to initiate restore"
		JSON(w, http.StatusInternalServerError, createRestoreResponse)
		return
	}

	createRestoreResponse.Success = true

	JSON(w, http.StatusOK, createRestoreResponse)
}

type RestoreAppsRequest struct {
	RestoreAll bool     `json:"restoreAll"`
	AppSlugs   []string `json:"appSlugs"`
}

type RestoreAppsResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) RestoreApps(w http.ResponseWriter, r *http.Request) {
	restoreResponse := RestoreAppsResponse{
		Success: false,
	}

	restoreAppsRequest := RestoreAppsRequest{}
	if err := json.NewDecoder(r.Body).Decode(&restoreAppsRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	snapshotName := mux.Vars(r)["snapshotName"]
	kotsadmNamespace := util.PodNamespace

	backup, err := snapshot.GetBackup(r.Context(), kotsadmNamespace, snapshotName)
	if err != nil {
		logger.Error(err)
		restoreResponse.Error = "failed to find backup"
		JSON(w, http.StatusInternalServerError, restoreResponse)
		return
	}

	if backup.Annotations[snapshottypes.InstanceBackupAnnotation] != "true" {
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
		restoreThisApp := false
		if !restoreAppsRequest.RestoreAll {
			for _, slug := range restoreAppsRequest.AppSlugs {
				if slug == a.Slug {
					restoreThisApp = true
					break
				}
			}
		}

		if !restoreThisApp && !restoreAppsRequest.RestoreAll {
			continue
		}

		if err := app.ResetRestore(a.ID); err != nil {
			logger.Error(err)
			restoreResponse.Error = fmt.Sprintf("failed to reset restore for app %s", a.Slug)
			JSON(w, http.StatusInternalServerError, restoreResponse)
			return
		}

		restoreName := fmt.Sprintf("%s.%s", snapshotName, a.Slug)
		if err := snapshot.DeleteRestore(r.Context(), kotsadmNamespace, restoreName); err != nil {
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

type GetRestoreAppsStatusRequest struct {
	CheckAll bool     `json:"checkAll"`
	AppSlugs []string `json:"appSlugs"`
}
type GetRestoreAppsStatusResponse struct {
	Statuses []AppRestoreStatus `json:"statuses"`
	Error    string             `json:"error,omitempty"`
}
type AppRestoreStatus struct {
	AppSlug       string                      `json:"appSlug"`
	RestoreDetail snapshottypes.RestoreDetail `json:"restoreDetail"`
}

func (h *Handler) GetRestoreAppsStatus(w http.ResponseWriter, r *http.Request) {
	response := GetRestoreAppsStatusResponse{
		Statuses: []AppRestoreStatus{},
	}

	restoreAppStatusRequest := GetRestoreAppsStatusRequest{}
	if err := json.NewDecoder(r.Body).Decode(&restoreAppStatusRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	snapshotName := mux.Vars(r)["snapshotName"]
	kotsadmNamespace := util.PodNamespace

	backup, err := snapshot.GetBackup(r.Context(), kotsadmNamespace, snapshotName)
	if err != nil {
		logger.Error(err)
		response.Error = "failed to find backup"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if backup.Annotations[snapshottypes.InstanceBackupAnnotation] != "true" {
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
		checkThisApp := false
		if !restoreAppStatusRequest.CheckAll {
			for _, slug := range restoreAppStatusRequest.AppSlugs {
				if slug == a.Slug {
					checkThisApp = true
					break
				}
			}
		}
		if !checkThisApp && !restoreAppStatusRequest.CheckAll {
			continue
		}

		restoreName := fmt.Sprintf("%s.%s", snapshotName, a.Slug)
		restoreDetail, err := snapshot.GetRestoreDetails(r.Context(), kotsadmNamespace, restoreName)
		if err != nil {
			if !kuberneteserrors.IsNotFound(errors.Cause(err)) {
				logger.Error(err)
				response.Error = fmt.Sprintf("failed to get restore for app %s", a.Slug)
				JSON(w, http.StatusInternalServerError, response)
				return
			}
		}

		if restoreDetail == nil {
			restoreDetail = &snapshottypes.RestoreDetail{
				Name:  restoreName,
				Phase: velerov1.RestorePhaseNew,
			}
		}

		appRestoreStatus := AppRestoreStatus{
			AppSlug:       a.Slug,
			RestoreDetail: *restoreDetail,
		}

		statuses = append(statuses, appRestoreStatus)
	}

	response.Statuses = statuses

	JSON(w, http.StatusOK, response)
}

func (h *Handler) GetRestoreStatus(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) CancelRestore(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) GetRestoreDetails(w http.ResponseWriter, r *http.Request) {
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

	restoreDetail, err := snapshot.GetRestoreDetails(r.Context(), util.PodNamespace, restoreName)
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
				Phase: velerov1.RestorePhaseFailed,
				Errors: []snapshottypes.SnapshotError{{
					Title:   "Restore has failed",
					Message: "Please check logs for errors.",
				}},
			}
		} else {
			restoreDetail = &snapshottypes.RestoreDetail{
				Name:  restoreName,
				Phase: velerov1.RestorePhaseNew,
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
