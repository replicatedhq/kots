package handlers

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/updatechecker"
	"github.com/replicatedhq/kots/pkg/util"
)

type AppUpdateCheckRequest struct {
}

type AppUpdateCheckResponse struct {
	AvailableUpdates int64 `json:"availableUpdates"`
}

func AppUpdateCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		w.WriteHeader(401)
		return
	}

	foundApp, err := app.GetFromSlug(mux.Vars(r)["appSlug"])
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
		AvailableUpdates: availableUpdates,
	}

	JSON(w, 200, appUpdateCheckResponse)
}
