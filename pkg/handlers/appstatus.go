package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/replicatedhq/kots/pkg/api/appstatus/types"
	"github.com/replicatedhq/kots/pkg/appstatus"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
)

// NOTE: this uses special cluster authorization
func (h *Handler) SetAppStatus(w http.ResponseWriter, r *http.Request) {
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

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newAppStatus := types.AppStatus{}
	err = json.Unmarshal(body, &newAppStatus)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	currentAppStatus, err := store.GetStore().GetAppStatus(newAppStatus.AppID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = store.GetStore().SetAppStatus(newAppStatus.AppID, newAppStatus.ResourceStates, newAppStatus.UpdatedAt)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newAppState := appstatus.GetState(newAppStatus.ResourceStates)
	if currentAppStatus != nil && newAppState != currentAppStatus.State {
		go func() {
			if err := reporting.SendAppInfo(newAppStatus.AppID); err != nil {
				logger.Debugf("failed to send app info: %#v", err)
			}
		}()
	}

	w.WriteHeader(http.StatusNoContent)
}
