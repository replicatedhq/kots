package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
)

func GetApp(w http.ResponseWriter, r *http.Request) {
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
	foundApp, err := app.GetFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, foundApp)
}
