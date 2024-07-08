package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/tasks"
	"github.com/replicatedhq/kots/pkg/update"
	"github.com/replicatedhq/kots/pkg/upgradeservice"
	upgradeservicetask "github.com/replicatedhq/kots/pkg/upgradeservice/task"
	upgradeservicetypes "github.com/replicatedhq/kots/pkg/upgradeservice/types"
)

type StartUpgradeServiceRequest struct {
	VersionLabel string `json:"versionLabel"`
	UpdateCursor string `json:"updateCursor"`
	ChannelID    string `json:"channelId"`
}

type StartUpgradeServiceResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type GetUpgradeServiceStatusResponse struct {
	CurrentMessage string `json:"currentMessage"`
	Status         string `json:"status"`
}

func (h *Handler) StartUpgradeService(w http.ResponseWriter, r *http.Request) {
	response := StartUpgradeServiceResponse{
		Success: false,
	}

	request := StartUpgradeServiceRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error = "failed to decode request body"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		response.Error = "failed to get app from app slug"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	canStart, reason, err := canStartUpgradeService(foundApp, request)
	if err != nil {
		response.Error = "failed to check if upgrade service can start"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}
	if !canStart {
		response.Error = reason
		logger.Error(errors.New(response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	if err := startUpgradeService(foundApp, request); err != nil {
		response.Error = "failed to start upgrade service"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	response.Success = true

	JSON(w, http.StatusOK, response)
}

func (h *Handler) GetUpgradeServiceStatus(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	status, message, err := upgradeservicetask.GetStatus(appSlug)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(err)
		return
	}

	JSON(w, http.StatusOK, GetUpgradeServiceStatusResponse{
		CurrentMessage: message,
		Status:         status,
	})
}

func canStartUpgradeService(a *apptypes.App, r StartUpgradeServiceRequest) (bool, string, error) {
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
		if currLicense.Spec.ChannelID != airgap.Spec.ChannelID || r.ChannelID != airgap.Spec.ChannelID {
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

	ll, err := replicatedapp.GetLatestLicense(currLicense)
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

func startUpgradeService(a *apptypes.App, r StartUpgradeServiceRequest) error {
	if err := upgradeservicetask.SetStatusStarting(a.Slug, "Preparing..."); err != nil {
		return errors.Wrap(err, "failed to set upgrade service task status")
	}

	go func() (finalError error) {
		finishedChan := make(chan error)
		defer close(finishedChan)

		tasks.StartTaskMonitor(upgradeservicetask.GetID(a.Slug), finishedChan)
		defer func() {
			if finalError != nil {
				logger.Error(finalError)
			}
			finishedChan <- finalError
		}()

		params, err := getUpgradeServiceParams(a, r)
		if err != nil {
			return err
		}
		if err := upgradeservice.Start(*params); err != nil {
			return errors.Wrap(err, "failed to start upgrade service")
		}
		return nil
	}()

	return nil
}

func getUpgradeServiceParams(a *apptypes.App, r StartUpgradeServiceRequest) (*upgradeservicetypes.UpgradeServiceParams, error) {
	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get registry details for app")
	}

	baseArchive, baseSequence, err := store.GetStore().GetAppVersionBaseArchive(a.ID, r.VersionLabel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version base archive")
	}

	nextSequence, err := store.GetStore().GetNextAppSequence(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get next app sequence")
	}

	source := "Upstream Update"
	if a.IsAirgap {
		source = "Airgap Update"
	}

	license, err := kotsutil.LoadLicenseFromBytes([]byte(a.License))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse app license")
	}

	var updateKOTSVersion string
	var updateKOTSBin string
	var updateAirgapBundle string

	if a.IsAirgap {
		au, err := update.GetAirgapUpdate(a.Slug, r.ChannelID, r.UpdateCursor)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get airgap update")
		}
		updateAirgapBundle = au
		kb, err := archives.GetKOTSBinFromAirgapBundle(au)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get kots binary from airgap bundle")
		}
		updateKOTSBin = kb
		kv, err := kotsutil.GetKOTSVersionFromBinary(kb)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get kots version from binary")
		}
		updateKOTSVersion = kv
	} else {
		kv, err := replicatedapp.GetKOTSVersionForRelease(license, r.VersionLabel)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get kots version for release")
		}
		updateKOTSVersion = kv

		if kv == buildversion.Version() {
			updateKOTSBin = kotsutil.GetKOTSBinPath()
		} else {
			kb, err := replicatedapp.DownloadKOTSBinary(license, r.VersionLabel)
			if err != nil {
				return nil, errors.Wrap(err, "failed to download kots binary")
			}
			updateKOTSBin = kb
		}
	}

	port, err := freeport.GetFreePort()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get free port")
	}

	return &upgradeservicetypes.UpgradeServiceParams{
		Port: fmt.Sprintf("%d", port),

		AppID:       a.ID,
		AppSlug:     a.Slug,
		AppName:     a.Name,
		AppIsAirgap: a.IsAirgap,
		AppIsGitOps: a.IsGitOps,
		AppLicense:  a.License,
		AppArchive:  baseArchive,

		Source:       source,
		BaseSequence: baseSequence,
		NextSequence: nextSequence,

		UpdateVersionLabel: r.VersionLabel,
		UpdateCursor:       r.UpdateCursor,
		UpdateChannelID:    r.ChannelID,
		UpdateAirgapBundle: updateAirgapBundle,

		CurrentKOTSVersion: buildversion.Version(),
		UpdateKOTSVersion:  updateKOTSVersion,
		UpdateKOTSBin:      updateKOTSBin,

		RegistryEndpoint:   registrySettings.Hostname,
		RegistryUsername:   registrySettings.Username,
		RegistryPassword:   registrySettings.Password,
		RegistryNamespace:  registrySettings.Namespace,
		RegistryIsReadOnly: registrySettings.IsReadOnly,

		ReportingInfo: reporting.GetReportingInfo(a.ID),
	}, nil
}
