package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/logger"
)

type UpdateAppRegistryRequest struct {
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Namespace string `json:"namespace"`
}

type UpdateAppRegistryResponse struct {
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
	Namespace string `json:"namespace"`
}

func UpdateAppRegistry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	updateAppRegistryRequest := UpdateAppRegistryRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateAppRegistryRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	foundApp, err := app.GetFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	err = app.UpdateRegistry(foundApp.ID, updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, updateAppRegistryRequest.Password, updateAppRegistryRequest.Namespace)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	updateAppRegistryResponse := UpdateAppRegistryResponse{
		Hostname:  updateAppRegistryRequest.Hostname,
		Username:  updateAppRegistryRequest.Username,
		Namespace: updateAppRegistryRequest.Namespace,
	}

	// if hostname and namespace have not changed, we don't need to re-push
	if foundApp.RegistrySettings != nil {
		if foundApp.RegistrySettings.Hostname == updateAppRegistryRequest.Hostname {
			if foundApp.RegistrySettings.Namespace == updateAppRegistryRequest.Namespace {
				JSON(w, 200, updateAppRegistryResponse)
				return
			}
		}
	}

	// in a goroutine, start pushing the images to the remote registry
	// we will let this function return while this happens
	go func() {
		if err := foundApp.RewriteImages(updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, updateAppRegistryRequest.Password,
			updateAppRegistryRequest.Namespace, nil); err != nil {
			logger.Error(err)
		}

	}()

	JSON(w, 200, updateAppRegistryResponse)
}

func GetAppRegistry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

}
