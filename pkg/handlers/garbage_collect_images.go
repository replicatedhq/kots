package handlers

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
)

type GarbageCollectImagesResponse struct {
	Error string `json:"error,omitempty"`
}

func (h *Handler) GarbageCollectImages(w http.ResponseWriter, r *http.Request) {
	response := GarbageCollectImagesResponse{}

	installParams, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
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

	isKurl, err := kotsadm.IsKurl()
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
			err := deleteUnusedImages(app.ID)
			if err != nil {
				if _, ok := err.(appRollbackError); ok {
					logger.Infof("not garbage collecting images because version allows rollbacks: %v", err)
				} else {
					logger.Error(errors.Wrap(err, "failed to delete unused images"))
				}
			}
		}
	}()

	JSON(w, http.StatusOK, response)
}
