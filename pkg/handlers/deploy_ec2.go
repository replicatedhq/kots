package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/plan"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/update"
)

type DeployEC2AppVersionRequest struct {
	VersionLabel string `json:"versionLabel"`
	UpdateCursor string `json:"updateCursor"`
	ChannelID    string `json:"channelId"`
}

type DeployEC2AppVersionResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) DeployEC2AppVersion(w http.ResponseWriter, r *http.Request) {
	response := DeployEC2AppVersionResponse{
		Success: false,
	}

	request := DeployEC2AppVersionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error = "failed to decode request body"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		response.Error = "failed to get app from slug"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	canDeploy, reason, err := canDeployEC2AppVersion(foundApp, request)
	if err != nil {
		response.Error = "failed to check if upgrade service can start"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}
	if !canDeploy {
		response.Error = reason
		logger.Error(errors.New(response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	p, err := plan.PlanUpgrade(store.GetStore(), plan.PlanUpgradeOptions{
		AppSlug:      appSlug,
		VersionLabel: request.VersionLabel,
		UpdateCursor: request.UpdateCursor,
		ChannelID:    request.ChannelID,
	})
	if err != nil {
		response.Error = "failed to plan upgrade"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if err := store.GetStore().UpsertPlan(p); err != nil {
		response.Error = "failed to upsert plan"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	go func() {
		if err := plan.Execute(store.GetStore(), p); err != nil {
			logger.Error(errors.Wrap(err, "failed to execute upgrade plan"))
		}
	}()

	response.Success = true

	JSON(w, http.StatusOK, response)
}

func canDeployEC2AppVersion(a *apptypes.App, r DeployEC2AppVersionRequest) (bool, string, error) {
	currLicense, err := kotsutil.LoadLicenseFromBytes([]byte(a.License))
	if err != nil {
		return false, "", errors.Wrap(err, "failed to parse app license")
	}

	if a.IsAirgap {
		updateBundle, err := update.GetAirgapUpdate(a.Slug, r.ChannelID, r.UpdateCursor)
		if err != nil {
			return false, "", errors.Wrap(err, "failed to get airgap update")
		}
		airgap, err := kotsutil.FindAirgapMetaInBundle(updateBundle)
		if err != nil {
			return false, "", errors.Wrap(err, "failed to find airgap metadata")
		}
		if _, err := kotsutil.FindChannelInLicense(airgap.Spec.ChannelID, currLicense); err != nil {
			return false, "channel mismatch, channel not in license", nil
		}
		if r.ChannelID != airgap.Spec.ChannelID {
			return false, "channel mismatch", nil
		}
		isDeployable, nonDeployableCause, err := update.IsAirgapUpdateDeployable(a, airgap)
		if err != nil {
			return false, "", errors.Wrap(err, "failed to check if airgap update is deployable")
		}
		if !isDeployable {
			return false, nonDeployableCause, nil
		}
		return true, "", nil
	}

	ll, err := replicatedapp.GetLatestLicense(currLicense, a.SelectedChannelID)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get latest license")
	}
	if currLicense.Spec.ChannelID != ll.License.Spec.ChannelID || r.ChannelID != ll.License.Spec.ChannelID {
		return false, "license channel has changed, please sync the license", nil
	}
	updates, err := update.GetAvailableUpdates(store.GetStore(), a, currLicense)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get available updates")
	}
	isDeployable, nonDeployableCause := false, "update not found"
	for _, u := range updates {
		if u.UpdateCursor == r.UpdateCursor {
			isDeployable, nonDeployableCause = u.IsDeployable, u.NonDeployableCause
			break
		}
	}
	if !isDeployable {
		return false, nonDeployableCause, nil
	}
	return true, "", nil
}