package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/updatechecker"
	cron "github.com/robfig/cron/v3"
)

type UpdateCheckerSpecRequest struct {
	UpdateCheckerSpec string `json:"updateCheckerSpec"`
}

type UpdateCheckerSpecResponse struct {
	Error string `json:"error"`
}

func UpdateCheckerSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	updateCheckerSpecResponse := &UpdateCheckerSpecResponse{}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to parse authorization header"
		JSON(w, 401, updateCheckerSpecResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		updateCheckerSpecResponse.Error = "failed to parse authorization header"
		JSON(w, 401, updateCheckerSpecResponse)
		return
	}

	updateCheckerSpecRequest := UpdateCheckerSpecRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateCheckerSpecRequest); err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to decode request body"
		JSON(w, 400, updateCheckerSpecResponse)
		return
	}

	foundApp, err := app.GetFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to get app from slug"
		JSON(w, 500, updateCheckerSpecResponse)
		return
	}

	if foundApp.IsAirgap {
		logger.Error(errors.New("airgap scheduled update checks are not supported"))
		updateCheckerSpecResponse.Error = "airgap scheduled update checks are not supported"
		JSON(w, 400, updateCheckerSpecResponse)
		return
	}

	// validate cron spec
	cronSpec := updateCheckerSpecRequest.UpdateCheckerSpec
	if cronSpec != "@never" {
		_, err := cron.ParseStandard(cronSpec)
		if err != nil {
			logger.Error(err)
			updateCheckerSpecResponse.Error = "failed to parse cron spec"
			JSON(w, 500, updateCheckerSpecResponse)
			return
		}
	}

	if err := app.SetUpdateCheckerSpec(foundApp.ID, cronSpec); err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to set update checker spec"
		JSON(w, 500, updateCheckerSpecResponse)
		return
	}

	// reconfigure update checker for the app
	if err := updatechecker.Configure(foundApp.ID); err != nil {
		logger.Error(err)
		updateCheckerSpecResponse.Error = "failed to reconfigure update checker cron job"
		JSON(w, 500, updateCheckerSpecResponse)
		return
	}

	JSON(w, 204, "")
}
