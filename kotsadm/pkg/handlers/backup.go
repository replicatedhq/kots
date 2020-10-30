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

type CreateApplicationBackupRequest struct {
}

type CreateApplicationBackupResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func CreateApplicationBackup(w http.ResponseWriter, r *http.Request) {
	createApplicationBackupResponse := CreateApplicationBackupResponse{
		Success: false,
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		createApplicationBackupResponse.Error = "failed to get app from app slug"
		JSON(w, 500, createApplicationBackupResponse)
		return
	}

	_, err = snapshot.CreateApplicationBackup(context.TODO(), foundApp, false)
	if err != nil {
		logger.Error(err)
		createApplicationBackupResponse.Error = "failed to create backup"
		JSON(w, 500, createApplicationBackupResponse)
		return
	}

	createApplicationBackupResponse.Success = true

	JSON(w, 200, createApplicationBackupResponse)
}

type ListBackupsResponse struct {
	Error   string                  `json:"error,omitempty"`
	Backups []*snapshottypes.Backup `json:"backups"`
}

func ListBackups(w http.ResponseWriter, r *http.Request) {
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

type ListInstanceBackupsResponse struct {
	Error   string                  `json:"error,omitempty"`
	Backups []*snapshottypes.Backup `json:"backups"`
}

func ListInstanceBackups(w http.ResponseWriter, r *http.Request) {
	listBackupsResponse := ListInstanceBackupsResponse{}

	backups, err := snapshot.ListInstanceBackups()
	if err != nil {
		logger.Error(err)
		listBackupsResponse.Error = "failed to list instance backups"
		JSON(w, 500, listBackupsResponse)
		return
	}
	listBackupsResponse.Backups = backups

	JSON(w, 200, listBackupsResponse)
}

type GetBackupResponse struct {
	BackupDetail *snapshottypes.BackupDetail `json:"backupDetail"`
	Success      bool                        `json:"success"`
	Error        string                      `json:"error,omitempty"`
}

func GetBackup(w http.ResponseWriter, r *http.Request) {
	getBackupResponse := GetBackupResponse{}

	backup, err := snapshot.GetBackupDetail(context.TODO(), mux.Vars(r)["snapshotName"])
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

type DeleteBackupResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func DeleteBackup(w http.ResponseWriter, r *http.Request) {
	deleteBackupResponse := DeleteBackupResponse{}

	if err := snapshot.DeleteBackup(mux.Vars(r)["snapshotName"]); err != nil {
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

func CreateInstanceBackup(w http.ResponseWriter, r *http.Request) {
	createInstanceBackupResponse := CreateInstanceBackupResponse{
		Success: false,
	}

	backup, err := snapshot.CreateInstanceBackup(context.TODO(), false)
	if err != nil {
		logger.Error(err)
		createInstanceBackupResponse.Error = "failed to create instance backup"
		JSON(w, http.StatusInternalServerError, createInstanceBackupResponse)
		return
	}

	createInstanceBackupResponse.Success = true
	createInstanceBackupResponse.BackupName = backup.ObjectMeta.Name

	JSON(w, 200, createInstanceBackupResponse)
}
