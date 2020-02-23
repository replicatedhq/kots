package handlers

import (
	"net/http"

	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/logger"
)

type CreateAppFromAirgapRequest struct {
}

type CreateAppFromAirgapResponse struct {
}

func CreateAppFromAirgap(w http.ResponseWriter, r *http.Request) {
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

	pendingApp, err := app.GetPendingAirgapUploadApp()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	registryHost := r.FormValue("registryHost")
	namespace := r.FormValue("namespace")
	username := r.FormValue("username")
	password := r.FormValue("password")

	airgapBundle, _, err := r.FormFile("file")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	go func() {
		defer airgapBundle.Close()
		if err := app.CreateAppFromAirgap(pendingApp, airgapBundle, registryHost, namespace, username, password); err != nil {
			logger.Error(err)
		}
	}()

	createAppFromAirgapResponse := CreateAppFromAirgapResponse{}

	JSON(w, 202, createAppFromAirgapResponse)
}
