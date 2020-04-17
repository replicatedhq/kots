package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/session"
	"github.com/replicatedhq/kotsadm/pkg/snapshot"
)

type CreateRestoreRequest struct {
}

type CreateRestoreResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func CreateRestore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	createRestoreResponse := CreateRestoreResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to parse authorization header"
		JSON(w, 401, createRestoreResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		createRestoreResponse.Error = "failed to parse authorization header"
		JSON(w, 401, createRestoreResponse)
		return
	}

	err = snapshot.CreateRestore(mux.Vars(r)["snapshotName"])
	if err != nil {
		logger.Error(err)
		createRestoreResponse.Error = "failed to create restore"
		JSON(w, 500, createRestoreResponse)
		return
	}

	createRestoreResponse.Success = true

	JSON(w, 200, createRestoreResponse)
}
