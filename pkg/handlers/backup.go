package handlers

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	"github.com/replicatedhq/kots/pkg/logger"
	kotssnapshot "github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
)

type CreateApplicationBackupRequest struct {
}

type CreateApplicationBackupResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type VeleroRBACResponse struct {
	Success                     bool   `json:"success"`
	Error                       string `json:"error,omitempty"`
	KotsadmNamespace            string `json:"kotsadmNamespace,omitempty"`
	KotsadmRequiresVeleroAccess bool   `json:"kotsadmRequiresVeleroAccess,omitempty"`
}

func (h *Handler) CreateApplicationBackup(w http.ResponseWriter, r *http.Request) {
	createApplicationBackupResponse := CreateApplicationBackupResponse{
		Success: false,
	}

	if util.IsEmbeddedCluster() {
		createApplicationBackupResponse.Error = "application backups are not supported in embedded clusters"
		JSON(w, http.StatusForbidden, createApplicationBackupResponse)
		return
	}

	// check minimal rbac
	if err := requiresKotsadmVeleroAccess(w, r); err != nil {
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app from slug"))
		createApplicationBackupResponse.Error = "failed to get app from app slug"
		JSON(w, http.StatusInternalServerError, createApplicationBackupResponse)
		return
	}

	_, err = snapshot.CreateApplicationBackup(r.Context(), foundApp, false)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create application snapshot"))
		createApplicationBackupResponse.Error = "failed to create backup"
		JSON(w, http.StatusInternalServerError, createApplicationBackupResponse)
		return
	}

	createApplicationBackupResponse.Success = true

	JSON(w, http.StatusOK, createApplicationBackupResponse)
}

type ListBackupsResponse struct {
	Error   string                  `json:"error,omitempty"`
	Backups []*snapshottypes.Backup `json:"backups"`
}

func (h *Handler) ListBackups(w http.ResponseWriter, r *http.Request) {
	listBackupsResponse := ListBackupsResponse{}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		listBackupsResponse.Error = "failed to get app from app slug"
		JSON(w, http.StatusInternalServerError, listBackupsResponse)
		return
	}

	kotsadmNamespace := util.PodNamespace

	veleroStatus, err := kotssnapshot.DetectVelero(r.Context(), kotsadmNamespace)
	if err != nil {
		logger.Error(err)
		listBackupsResponse.Error = "failed to detect velero"
		JSON(w, http.StatusInternalServerError, listBackupsResponse)
		return
	}

	if veleroStatus == nil {
		JSON(w, http.StatusOK, listBackupsResponse)
		return
	}

	backups, err := snapshot.ListBackupsForApp(r.Context(), kotsadmNamespace, foundApp.ID)
	if err != nil {
		logger.Error(err)
		listBackupsResponse.Error = "failed to list backups"
		JSON(w, http.StatusInternalServerError, listBackupsResponse)
		return
	}
	listBackupsResponse.Backups = backups

	JSON(w, http.StatusOK, listBackupsResponse)
}

type ListInstanceBackupsResponse struct {
	Error   string                            `json:"error,omitempty"`
	Backups []*snapshottypes.ReplicatedBackup `json:"backups"`
}

func (h *Handler) ListInstanceBackups(w http.ResponseWriter, r *http.Request) {
	listBackupsResponse := ListInstanceBackupsResponse{}

	replicatedBackups, err := snapshot.ListInstanceBackups(r.Context(), util.PodNamespace)
	if err != nil {
		logger.Error(err)
		listBackupsResponse.Error = "failed to list instance backups"
		JSON(w, http.StatusInternalServerError, listBackupsResponse)
		return
	}
	listBackupsResponse.Backups = replicatedBackups

	JSON(w, http.StatusOK, listBackupsResponse)
}

type GetBackupResponse struct {
	BackupDetail *snapshottypes.BackupDetail `json:"backupDetail"`
	Success      bool                        `json:"success"`
	Error        string                      `json:"error,omitempty"`
}

func (h *Handler) GetBackup(w http.ResponseWriter, r *http.Request) {
	getBackupResponse := GetBackupResponse{}

	backup, err := snapshot.GetBackupDetail(r.Context(), util.PodNamespace, mux.Vars(r)["snapshotName"])
	if err != nil {
		logger.Error(err)
		getBackupResponse.Error = "failed to get backup detail"
		JSON(w, 500, getBackupResponse)
		return
	}
	getBackupResponse.BackupDetail = backup

	getBackupResponse.Success = true

	JSON(w, http.StatusOK, getBackupResponse)
}

type DeleteBackupResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) DeleteBackup(w http.ResponseWriter, r *http.Request) {
	deleteBackupResponse := DeleteBackupResponse{}

	if err := snapshot.DeleteBackup(r.Context(), util.PodNamespace, mux.Vars(r)["snapshotName"]); err != nil {
		logger.Error(err)
		deleteBackupResponse.Error = "failed to delete backup"
		JSON(w, http.StatusInternalServerError, deleteBackupResponse)
		return
	}

	deleteBackupResponse.Success = true

	JSON(w, http.StatusOK, deleteBackupResponse)
}

type CreateInstanceBackupRequest struct {
}

type CreateInstanceBackupResponse struct {
	Success    bool   `json:"success"`
	BackupName string `json:"backupName,omitempty"`
	Error      string `json:"error,omitempty"`
}

func (h *Handler) CreateInstanceBackup(w http.ResponseWriter, r *http.Request) {
	createInstanceBackupResponse := CreateInstanceBackupResponse{
		Success: false,
	}

	// check minimal rbac
	if err := requiresKotsadmVeleroAccess(w, r); err != nil {
		return
	}

	clusters, err := store.GetStore().ListClusters()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list clusters"))
		createInstanceBackupResponse.Error = "failed to list clusters"
		JSON(w, http.StatusInternalServerError, createInstanceBackupResponse)
		return
	}
	if len(clusters) == 0 {
		logger.Error(errors.New("No clusters found"))
		createInstanceBackupResponse.Error = "no clusters found"
		JSON(w, http.StatusInternalServerError, createInstanceBackupResponse)
		return
	}
	c := clusters[0]

	backupName, err := snapshot.CreateInstanceBackup(context.TODO(), c, false)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create instance snapshot"))
		createInstanceBackupResponse.Error = "failed to create instance backup"
		JSON(w, http.StatusInternalServerError, createInstanceBackupResponse)
		return
	}

	createInstanceBackupResponse.Success = true
	createInstanceBackupResponse.BackupName = backupName

	JSON(w, http.StatusOK, createInstanceBackupResponse)
}
