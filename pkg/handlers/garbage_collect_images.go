package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/registry"
	"github.com/replicatedhq/kots/pkg/store"
)

type GarbageCollectImagesRequest struct {
	IgnoreRollback bool `json:"ignoreRollback,omitempty"`
}

type GarbageCollectImagesResponse struct {
	Error string `json:"error,omitempty"`
}

func (h *Handler) GarbageCollectImages(w http.ResponseWriter, r *http.Request) {
	response := GarbageCollectImagesResponse{}

	garbageCollectImagesRequest := GarbageCollectImagesRequest{}
	if err := json.NewDecoder(r.Body).Decode(&garbageCollectImagesRequest); err != nil {
		response.Error = "failed to decode request"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	installParams, err := kotsutil.GetInstallationParams()
	if err != nil {
		response.Error = "failed to get app registry info"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}
	if !installParams.EnableImageDeletion {
		response.Error = "image garbage collection is not enabled"
		logger.Error(errors.New(response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		response.Error = "failed to get k8s clientset"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	isKurl, err := kurl.IsKurl(clientset)
	if err != nil {
		response.Error = "failed to check kURL"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if !isKurl {
		response.Error = "image garbage collection is only supported in embedded clusters"
		logger.Error(errors.New(response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		response.Error = "failed to list apps"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if len(apps) == 0 {
		response.Error = "no installed apps found"
		logger.Error(errors.New(response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	go func() {
		for _, app := range apps {
			logger.Infof("Deleting images for app %s", app.Slug)
			err := registry.DeleteUnusedImages(app.ID, garbageCollectImagesRequest.IgnoreRollback)
			if err != nil {
				if _, ok := err.(registry.AppRollbackError); ok {
					logger.Infof("not garbage collecting images because version allows rollbacks: %v", err)
				} else {
					logger.Error(errors.Wrap(err, "failed to delete unused images"))
				}
			}
		}
	}()

	JSON(w, http.StatusOK, response)
}
