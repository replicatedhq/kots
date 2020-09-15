package handlers

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

type GetDownstreamOutputResponse struct {
	Logs DownstreamLogs `json:"logs"`
}
type DownstreamLogs struct {
	DryrunStdout string `json:"dryrunStdout"`
	DryrunStderr string `json:"dryrunStderr"`
	ApplyStdout  string `json:"applyStdout"`
	ApplyStderr  string `json:"applyStderr"`
	RenderError  string `json:"renderError"`
}

func GetDownstreamOutput(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	clusterID := mux.Vars(r)["clusterId"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	output, err := downstream.GetDownstreamOutput(a.ID, clusterID, int64(sequence))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	downstreamLogs := DownstreamLogs{
		DryrunStdout: output.DryrunStdout,
		DryrunStderr: output.DryrunStderr,
		ApplyStdout:  output.ApplyStdout,
		ApplyStderr:  output.ApplyStderr,
		RenderError:  output.RenderError,
	}
	getDownstreamOutputResponse := GetDownstreamOutputResponse{
		Logs: downstreamLogs,
	}

	JSON(w, http.StatusOK, getDownstreamOutputResponse)
}
