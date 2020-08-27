package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/replicatedhq/kots/kotsadm/pkg/appstatus"
	"github.com/replicatedhq/kots/kotsadm/pkg/appstatus/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

// NOTE: this uses special cluster authorization
func SetAppStatus(w http.ResponseWriter, r *http.Request) {
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

	status := types.AppStatus{}
	err = json.Unmarshal(body, &status)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = appstatus.Set(status.AppID, status.ResourceStates, status.UpdatedAt)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
