package handlers

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshot"
	snapshottypes "github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

type CreateBackupRequest struct {
}

type CreateBackupResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func CreateBackup(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	createBackupResponse := CreateBackupResponse{
		Success: false,
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		createBackupResponse.Error = "failed to get app from app slug"
		JSON(w, 500, createBackupResponse)
		return
	}

	_, err = snapshot.CreateBackup(foundApp, false)
	if err != nil {
		logger.Error(err)
		createBackupResponse.Error = "failed to create backup"
		JSON(w, 500, createBackupResponse)
		return
	}

	createBackupResponse.Success = true

	JSON(w, 200, createBackupResponse)
}

type ListBackupsResponse struct {
	Error   string                  `json:"error,omitempty"`
	Backups []*snapshottypes.Backup `json:"backups"`
}

func ListBackups(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	listBackupsResponse := ListBackupsResponse{}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		listBackupsResponse.Error = "failed to get app from app slug"
		JSON(w, 500, listBackupsResponse)
		return
	}

	veleroStatus, err := snapshot.DetectVelero()
	if err != nil {
		logger.Error(err)
		listBackupsResponse.Error = "failed to detect velero"
		JSON(w, 500, listBackupsResponse)
		return
	}

	if veleroStatus == nil {
		JSON(w, 200, listBackupsResponse)
		return
	}

	backups, err := snapshot.ListBackupsForApp(foundApp.ID)
	if err != nil {
		logger.Error(err)
		listBackupsResponse.Error = "failed to list backups"
		JSON(w, 500, listBackupsResponse)
		return
	}
	listBackupsResponse.Backups = backups

	JSON(w, 200, listBackupsResponse)
}

type ListKotsadmBackupsResponse struct {
	Error   string                  `json:"error,omitempty"`
	Backups []*snapshottypes.Backup `json:"backups"`
}

func ListKotsadmBackups(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	listBackupsResponse := ListKotsadmBackupsResponse{}

	backups, err := snapshot.ListKotsadmBackups()
	if err != nil {
		logger.Error(err)
		listBackupsResponse.Error = "failed to list backups"
		JSON(w, 500, listBackupsResponse)
		return
	}
	listBackupsResponse.Backups = backups

	JSON(w, 200, listBackupsResponse)
}

type GetKotsadmBackupResponse struct {
	BackupDetail *snapshottypes.BackupDetail `json:"backupDetail"`
	Success      bool                        `json:"success"`
	Error        string                      `json:"error,omitempty"`
}

func GetKotsadmBackup(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	getBackupResponse := GetKotsadmBackupResponse{}

	backup, err := snapshot.GetKotsadmBackupDetail(context.TODO(), mux.Vars(r)["snapshotName"])
	if err != nil {
		logger.Error(err)
		getBackupResponse.Error = "failed to get backup detail"
		JSON(w, 500, getBackupResponse)
		return
	}
	getBackupResponse.BackupDetail = backup

	getBackupResponse.Success = true

	JSON(w, 200, getBackupResponse)
}

type DeleteKotsadmBackupResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func DeleteKotsadmBackup(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	deleteBackupResponse := DeleteKotsadmBackupResponse{}

	if err := snapshot.DeleteBackup(mux.Vars(r)["snapshotName"]); err != nil {
		logger.Error(err)
		deleteBackupResponse.Error = "failed to delete backup"
		JSON(w, http.StatusInternalServerError, deleteBackupResponse)
		return
	}

	deleteBackupResponse.Success = true

	JSON(w, http.StatusOK, deleteBackupResponse)
}
