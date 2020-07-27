package handlers

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
)

func DeployAppVersion(w http.ResponseWriter, r *http.Request) {
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

	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	a, err := app.GetFromSlug(appSlug)
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
