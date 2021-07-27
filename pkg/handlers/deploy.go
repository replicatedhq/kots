package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	versiontypes "github.com/replicatedhq/kots/pkg/api/version/types"
	"github.com/replicatedhq/kots/pkg/app"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/redact"
	"github.com/replicatedhq/kots/pkg/registry"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	"github.com/replicatedhq/kots/pkg/version"
	"go.uber.org/zap"
)

type UpdateDeployResultRequest struct {
	AppID        string `json:"appId"`
	IsError      bool   `json:"isError"`
	DryrunStdout string `json:"dryrunStdout"`
	DryrunStderr string `json:"dryrunStderr"`
	ApplyStdout  string `json:"applyStdout"`
	ApplyStderr  string `json:"applyStderr"`
	HelmStdout   string `json:"helmStdout"`
	HelmStderr   string `json:"helmStderr"`
	RenderError  string `json:"renderError"`
}

type UpdateUndeployResultRequest struct {
	AppID   string `json:"appId"`
	IsError bool   `json:"isError"`
}

type DeployAppVersionRequest struct {
	IsSkipPreflights             bool `json:"isSkipPreflights"`
	ContinueWithFailedPreflights bool `json:"continueWithFailedPreflights"`
	IsCLI                        bool `json:"isCli"`
}

func (h *Handler) DeployAppVersion(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	request := DeployAppVersionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.Error(errors.Wrap(err, "failed to decode request body"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse sequence number"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get app for slug %s", appSlug))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list downstreams for app"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if len(downstreams) == 0 {
		logger.Error(errors.Errorf("no downstreams for app %s", appSlug))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	status, err := store.GetStore().GetStatusForVersion(a.ID, downstreams[0].ClusterID, int64(sequence))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get update downstream status"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if status == storetypes.VersionPendingConfig {
		logger.Error(errors.Errorf("not deploying version %d because it's %s", int64(sequence), status))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := store.GetStore().DeleteDownstreamDeployStatus(a.ID, downstreams[0].ClusterID, int64(sequence)); err != nil {
		logger.Error(errors.Wrap(err, "failed to delete downstream deploy status"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := version.DeployVersion(a.ID, int64(sequence)); err != nil {
		logger.Error(errors.Wrap(err, "failed to queue version for deployment"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// preflights reports
	go func() {
		if request.IsSkipPreflights || request.ContinueWithFailedPreflights {
			if err := reporting.ReportAppInfo(a.ID, int64(sequence), request.IsSkipPreflights, request.IsCLI); err != nil {
				logger.Debugf("failed to send preflights data to replicated app: %v", err)
				return
			}
		}
	}()

	JSON(w, 204, "")
}

// NOTE: this uses special cluster authorization
func (h *Handler) UpdateDeployResult(w http.ResponseWriter, r *http.Request) {
	auth, err := parseClusterAuthorization(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse cluster auth"))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	clusterID, err := store.GetStore().GetClusterIDFromDeployToken(auth.Password)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get cluster ID"))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	updateDeployResultRequest := UpdateDeployResultRequest{}
	err = json.NewDecoder(r.Body).Decode(&updateDeployResultRequest)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to decode deploy result"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// sequence really should be passed down to operator and returned from it
	currentSequence, err := store.GetStore().GetCurrentSequence(updateDeployResultRequest.AppID, clusterID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get current sequence"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := createSupportBundleSpec(updateDeployResultRequest.AppID, currentSequence, "", true); err != nil {
		// support bundle is not essential.  keep processing deployment request
		logger.Error(errors.Wrapf(err, "failed to create support bundle for sequence %d after deploying", currentSequence))
	}

	alreadySuccessful, err := store.GetStore().IsDownstreamDeploySuccessful(updateDeployResultRequest.AppID, clusterID, currentSequence)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to check deploy successful"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if alreadySuccessful {
		w.WriteHeader(http.StatusOK)
		return
	}

	downstreamOutput := downstreamtypes.DownstreamOutput{
		DryrunStdout: updateDeployResultRequest.DryrunStdout,
		DryrunStderr: updateDeployResultRequest.DryrunStderr,
		ApplyStdout:  updateDeployResultRequest.ApplyStdout,
		ApplyStderr:  updateDeployResultRequest.ApplyStderr,
		HelmStdout:   updateDeployResultRequest.HelmStdout,
		HelmStderr:   updateDeployResultRequest.HelmStderr,
		RenderError:  updateDeployResultRequest.RenderError,
	}
	err = store.GetStore().UpdateDownstreamDeployStatus(updateDeployResultRequest.AppID, clusterID, currentSequence, updateDeployResultRequest.IsError, downstreamOutput)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to update downstream deploy status"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !updateDeployResultRequest.IsError {
		go func() {
			err := deleteUnusedImages(updateDeployResultRequest.AppID)
			if err != nil {
				if _, ok := err.(appRollbackError); ok {
					logger.Infof("not garbage collecting images because version allows rollbacks: %v", err)
				} else {
					logger.Infof("failed to delete unused images: %v", err)
				}
			}
		}()
	}

	w.WriteHeader(http.StatusOK)
	return
}

type appRollbackError struct {
	AppID    string
	Sequence int64
}

func (e appRollbackError) Error() string {
	return fmt.Sprintf("app:%s, version:%d", e.AppID, e.Sequence)
}

func deleteUnusedImages(appID string) error {
	installParams, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		return errors.Wrap(err, "failed to get app registry info")
	}
	if !installParams.EnableImageDeletion {
		return nil
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app registry info")
	}

	if registrySettings.IsReadOnly {
		return nil
	}

	isKurl, err := kotsadm.IsKurl()
	if err != nil {
		return errors.Wrap(err, "failed to check kURL")
	}

	if !isKurl {
		return nil
	}

	appIDs, err := store.GetStore().GetAppIDsFromRegistry(registrySettings.Hostname)
	if err != nil {
		return errors.Wrap(err, "failed to get apps with registry")
	}

	activeVersions := []*versiontypes.AppVersion{}
	for _, appID := range appIDs {
		downstreams, err := store.GetStore().ListDownstreamsForApp(appID)
		if err != nil {
			return errors.Wrap(err, "failed to list downstreams for app")
		}

		for _, d := range downstreams {
			curSequence, err := store.GetStore().GetCurrentParentSequence(appID, d.ClusterID)
			if err != nil {
				return errors.Wrap(err, "failed to get current parent sequence")
			}

			curVersion, err := store.GetStore().GetAppVersion(appID, curSequence)
			if err != nil {
				return errors.Wrap(err, "failed to get app version")
			}

			activeVersions = append(activeVersions, curVersion)

			laterVersions, err := store.GetStore().GetAppVersionsAfter(appID, curSequence)
			if err != nil {
				return errors.Wrapf(err, "failed to get versions after %d", curVersion.Sequence)
			}
			activeVersions = append(activeVersions, laterVersions...)
		}
	}

	imagesDedup := map[string]struct{}{}
	for _, version := range activeVersions {
		if version == nil {
			continue
		}
		if version.KOTSKinds == nil {
			continue
		}
		if version.KOTSKinds.KotsApplication.Spec.AllowRollback {
			return appRollbackError{AppID: version.AppID, Sequence: version.Sequence}
		}
		for _, i := range version.KOTSKinds.Installation.Spec.KnownImages {
			imagesDedup[i.Image] = struct{}{}
		}
	}

	usedImages := []string{}
	for i, _ := range imagesDedup {
		usedImages = append(usedImages, i)
	}

	if installParams.KotsadmRegistry != "" {
		deployOptions := kotsadmtypes.DeployOptions{
			// Minimal info needed to get the right image names
			KotsadmOptions: kotsadmtypes.KotsadmOptions{
				// TODO: OverrideVersion
				OverrideRegistry:  registrySettings.Hostname,
				OverrideNamespace: registrySettings.Namespace,
				Username:          registrySettings.Username,
				Password:          registrySettings.Password,
			},
		}
		kotsadmImages := kotsadmobjects.GetAdminConsoleImages(deployOptions)
		for _, i := range kotsadmImages {
			usedImages = append(usedImages, i)
		}
	}

	err = registry.DeleteUnusedImages(context.Background(), registrySettings, usedImages)
	if err != nil {
		return errors.Wrap(err, "failed to delete unused images")
	}

	return nil
}

func createSupportBundleSpec(appID string, sequence int64, origin string, inCluster bool) error {
	archivePath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archivePath)

	err = store.GetStore().GetAppVersionArchive(appID, sequence, archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to get current archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to load current kotskinds")
	}

	defaultOpts := supportbundle.DefaultTroubleshootOpts{
		Origin:    origin,
		InCluster: inCluster,
	}

	_, err = supportbundle.CreateRenderedSpec(appID, sequence, kotsKinds, defaultOpts)
	if err != nil {
		return errors.Wrap(err, "failed to create rendered support bundle spec")
	}

	err = redact.CreateRenderedAppRedactSpec(appID, sequence, kotsKinds)
	if err != nil {
		return errors.Wrap(err, "failed to write app redact spec configmap")
	}

	return nil
}

// NOTE: this uses special cluster authorization
func (h *Handler) UpdateUndeployResult(w http.ResponseWriter, r *http.Request) {
	auth, err := parseClusterAuthorization(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	_, err = store.GetStore().GetClusterIDFromDeployToken(auth.Password)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	updateUndeployResultRequest := UpdateUndeployResultRequest{}
	err = json.NewDecoder(r.Body).Decode(&updateUndeployResultRequest)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var status apptypes.UndeployStatus
	if updateUndeployResultRequest.IsError {
		status = apptypes.UndeployFailed
	} else {
		status = apptypes.UndeployCompleted
	}

	logger.Info("restore API set undeploy status",
		zap.String("status", string(status)),
		zap.String("appID", updateUndeployResultRequest.AppID))

	foundApp, err := store.GetStore().GetApp(updateUndeployResultRequest.AppID)
	if err != nil {
		err = errors.Wrap(err, "failed to get app")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if foundApp.RestoreInProgressName != "" {
		go func() {
			<-time.After(20 * time.Second)
			err = app.SetRestoreUndeployStatus(updateUndeployResultRequest.AppID, status)
			if err != nil {
				err = errors.Wrap(err, "failed to set app undeploy status")
				logger.Error(err)
				return
			}
		}()
	}

	w.WriteHeader(http.StatusOK)
	return
}
