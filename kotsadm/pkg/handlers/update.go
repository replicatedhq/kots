package handlers

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/updatechecker"
	"github.com/replicatedhq/kots/pkg/util"
)

type AppUpdateCheckRequest struct {
}

type AppUpdateCheckResponse struct {
	AvailableUpdates   int64 `json:"availableUpdates"`
	CurrentAppSequence int64 `json:"currentAppSequence"`
}

func AppUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	deploy := false
	d := r.URL.Query().Get("deploy")
	if d != "" {
		deploy, _ = strconv.ParseBool(d)
	}

	availableUpdates, err := updatechecker.CheckForUpdates(foundApp.ID, deploy)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		cause := errors.Cause(err)
		if _, ok := cause.(util.ActionableError); ok {
			w.Write([]byte(cause.Error()))
		}
		return
	}

	appUpdateCheckResponse := AppUpdateCheckResponse{
		AvailableUpdates:   availableUpdates,
		CurrentAppSequence: foundApp.CurrentSequence,
	}

	JSON(w, 200, appUpdateCheckResponse)
}
