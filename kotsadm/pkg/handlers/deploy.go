package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	"go.uber.org/zap"
)

type UpdateDeployResultRequest struct {
	AppID        string `json:"appId"`
	IsError      bool   `json:"isError"`
	DryrunStdout string `json:"dryrunStdout"`
	DryrunStderr string `json:"dryrunStderr"`
	ApplyStdout  string `json:"applyStdout"`
	ApplyStderr  string `json:"applyStderr"`
	RenderError  string `json:"renderError"`
}

type UpdateUndeployResultRequest struct {
	AppID   string `json:"appId"`
	IsError bool   `json:"isError"`
}

func DeployAppVersion(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := version.DeployVersion(a.ID, int64(sequence)); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	JSON(w, 204, "")
}

// NOTE: this uses special cluster authorization
func UpdateDeployResult(w http.ResponseWriter, r *http.Request) {
	auth, err := parseClusterAuthorization(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	clusterID, err := store.GetStore().GetClusterIDFromDeployToken(auth.Password)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	updateDeployResultRequest := UpdateDeployResultRequest{}
	err = json.NewDecoder(r.Body).Decode(&updateDeployResultRequest)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// sequence really should be passed down to operator and returned from it
	currentSequence, err := downstream.GetCurrentSequence(updateDeployResultRequest.AppID, clusterID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	alreadySuccessful, err := downstream.IsDownstreamDeploySuccessful(updateDeployResultRequest.AppID, clusterID, currentSequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if alreadySuccessful {
		w.WriteHeader(http.StatusOK)
		return
	}

	downstreamOutput := downstreamtypes.DownstreamOutput{
		DryrunStdout: updateDeployResultRequest.DryrunStdout,
		DryrunStderr: updateDeployResultRequest.DryrunStderr,
		ApplyStdout:  updateDeployResultRequest.ApplyStdout,
		ApplyStderr:  updateDeployResultRequest.ApplyStderr,
		RenderError:  updateDeployResultRequest.RenderError,
	}
	err = downstream.UpdateDownstreamDeployStatus(updateDeployResultRequest.AppID, clusterID, currentSequence, updateDeployResultRequest.IsError, downstreamOutput)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

// NOTE: this uses special cluster authorization
func UpdateUndeployResult(w http.ResponseWriter, r *http.Request) {
	auth, err := parseClusterAuthorization(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	_, err = store.GetStore().GetClusterIDFromDeployToken(auth.Password)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	updateUndeployResultRequest := UpdateUndeployResultRequest{}
	err = json.NewDecoder(r.Body).Decode(&updateUndeployResultRequest)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var status apptypes.UndeployStatus
	if updateUndeployResultRequest.IsError {
		status = apptypes.UndeployFailed
	} else {
		status = apptypes.UndeployCompleted
	}

	logger.Info("restore API set undeploy status",
		zap.String("status", string(status)),
		zap.String("appID", updateUndeployResultRequest.AppID))

	foundApp, err := store.GetStore().GetApp(updateUndeployResultRequest.AppID)
	if err != nil {
		err = errors.Wrap(err, "failed to get app")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if foundApp.RestoreInProgressName != "" {
		go func() {
			<-time.After(20 * time.Second)
			err = app.SetRestoreUndeployStatus(updateUndeployResultRequest.AppID, status)
			if err != nil {
				err = errors.Wrap(err, "failed to set app undeploy status")
				logger.Error(err)
				return
			}
		}()
	}

	w.WriteHeader(http.StatusOK)
	return
}
