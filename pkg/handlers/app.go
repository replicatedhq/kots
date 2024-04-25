package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/airgap"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/api/handlers/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/gitops"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator"
	"github.com/replicatedhq/kots/pkg/rbac"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/session"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/store/kotsstore"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

func (h *Handler) GetPendingApp(w http.ResponseWriter, r *http.Request) {
	sess := session.ContextGetSession(r)
	if sess == nil {
		logger.Error(errors.New("invalid session"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	papp, err := store.GetStore().GetPendingAirgapUploadApp()
	if err != nil {
		if store.GetStore().IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			logger.Error(errors.Wrap(err, "failed to get pending app"))
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	defaultRoles := rbac.DefaultRoles() // TODO (ethan): this should be set in the handler

	if sess.HasRBAC { // handle pre-rbac sessions
		allow, err := rbac.CheckAccess(r.Context(), defaultRoles, "read", fmt.Sprintf("app.%s", papp.Slug), sess.Roles)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to check access for pending app %s", papp.Slug))
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if !allow {
			logger.Debug("failed to check access for pending app")
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get clientset"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pendingAppResponse := types.GetPendingAppResponse{
		App: types.ResponsePendingApp{
			ID:            papp.ID,
			Slug:          papp.Slug,
			Name:          papp.Name,
			LicenseData:   papp.LicenseData,
			NeedsRegistry: !kotsutil.HasEmbeddedRegistry(clientset),
		},
	}
	JSON(w, http.StatusOK, pendingAppResponse)
}

func (h *Handler) ListApps(w http.ResponseWriter, r *http.Request) {
	sess := session.ContextGetSession(r)
	if sess == nil {
		logger.Error(errors.New("invalid session"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responseApps := []types.ResponseApp{}
	defaultRoles := rbac.DefaultRoles() // TODO (ethan): this should be set in the handler

	for _, a := range apps {
		if sess.HasRBAC { // handle pre-rbac sessions
			allow, err := rbac.CheckAccess(r.Context(), defaultRoles, "read", fmt.Sprintf("app.%s", a.Slug), sess.Roles)
			if err != nil {
				logger.Error(errors.Wrapf(err, "failed to check access for app %s", a.Slug))
				w.WriteHeader(http.StatusInternalServerError)
				return
			} else if !allow {
				continue
			}
		}

		responseApp, err := responseAppFromApp(a)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		responseApps = append(responseApps, *responseApp)
	}

	listAppsResponse := types.ListAppsResponse{
		Apps: responseApps,
	}

	JSON(w, http.StatusOK, listAppsResponse)
}

func (h *Handler) GetAppStatus(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	appStatus, err := store.GetStore().GetAppStatus(a.ID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	appStatusResponse := types.AppStatusResponse{
		AppStatus: appStatus,
	}
	JSON(w, http.StatusOK, appStatusResponse)
}

func (h *Handler) GetApp(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	responseApp, err := responseAppFromApp(a)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, responseApp)
}

func responseAppFromApp(a *apptypes.App) (*types.ResponseApp, error) {
	license, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get license")
	}

	appStatus, err := store.GetStore().GetAppStatus(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app status")
	}
	appState := string(appStatus.State)

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}
	if len(downstreams) == 0 {
		return nil, errors.New("no downstreams for app")
	}
	d := downstreams[0]

	appVersions, err := store.GetStore().GetDownstreamVersions(a.ID, d.ClusterID, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get downstream versions")
	}

	// "latest" is the version that can be viewed and configured (and possibly has been deployed)
	var latestVersion *downstreamtypes.DownstreamVersion
	for _, version := range appVersions.AllVersions {
		if version.Status != storetypes.VersionPendingDownload {
			latestVersion = version
			break
		}
	}

	if latestVersion == nil {
		// should never happen
		return nil, errors.Errorf("found %d versions, and 0 have been downloaded", len(appVersions.AllVersions))
	}

	isIdentityServiceSupportedForVersion, err := store.GetStore().IsIdentityServiceSupportedForVersion(a.ID, latestVersion.ParentSequence)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check if identity service is supported for version %d", latestVersion.ParentSequence)
	}
	isAppIdentityServiceSupported := isIdentityServiceSupportedForVersion && license.Spec.IsIdentityServiceSupported

	allowRollback, err := store.GetStore().IsRollbackSupportedForVersion(a.ID, latestVersion.ParentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if rollback is supported")
	}

	targetKotsVersion, err := store.GetStore().GetTargetKotsVersionForVersion(a.ID, latestVersion.ParentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get target kots version")
	}

	parentSequence, err := store.GetStore().GetCurrentParentSequence(a.ID, d.ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current parent sequence for downstream")
	}

	// check snapshots for the parent sequence of the deployed version
	s, err := store.GetStore().IsSnapshotsSupportedForVersion(a, parentSequence, &render.Renderer{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if snapshots is allowed")
	}
	allowSnapshots := s && license.Spec.IsSnapshotSupported

	isGitopsSupported := license.Spec.IsGitOpsSupported && !util.IsEmbeddedCluster() // gitops is not allowed in embedded cluster installations today

	links, err := version.GetRealizedLinksFromAppSpec(a.ID, parentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get realized links from app spec")
	}

	downstreamGitOps, err := gitops.GetDownstreamGitOps(a.ID, d.ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get downstream gitops")
	}
	responseGitOps := types.ResponseGitOps{}
	if downstreamGitOps != nil {
		responseGitOps = types.ResponseGitOps{
			Enabled:     true,
			Provider:    downstreamGitOps.Provider,
			Uri:         downstreamGitOps.RepoURI,
			Hostname:    downstreamGitOps.Hostname,
			HTTPPort:    downstreamGitOps.HTTPPort,
			SSHPort:     downstreamGitOps.SSHPort,
			Path:        downstreamGitOps.Path,
			Branch:      downstreamGitOps.Branch,
			Format:      downstreamGitOps.Format,
			Action:      downstreamGitOps.Action,
			DeployKey:   downstreamGitOps.PublicKey,
			IsConnected: downstreamGitOps.IsConnected,
		}
	}

	cluster := types.ResponseCluster{
		ID:   d.ClusterID,
		Slug: d.ClusterSlug,
	}

	if util.IsEmbeddedCluster() {
		var embeddedClusterConfig *embeddedclusterv1beta1.Config
		if appVersions.CurrentVersion != nil {
			embeddedClusterConfig, err = store.GetStore().GetEmbeddedClusterConfigForVersion(a.ID, appVersions.CurrentVersion.Sequence)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get embedded cluster config")
			}
		}

		if embeddedClusterConfig != nil {
			cluster.RequiresUpgrade, err = embeddedcluster.RequiresUpgrade(context.TODO(), embeddedClusterConfig.Spec)
			if err != nil {
				return nil, errors.Wrap(err, "failed to check if cluster requires upgrade")
			}

			embeddedClusterInstallations, err := embeddedcluster.ListInstallations(context.TODO())
			if err != nil {
				return nil, errors.Wrap(err, "failed to list installations")
			}

			cluster.NumInstallations = len(embeddedClusterInstallations)

			currentInstallation, err := embeddedcluster.GetCurrentInstallation(context.TODO())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get latest installation")
			}

			if currentInstallation != nil {
				cluster.State = string(currentInstallation.Status.State)
			}
		}
	}

	responseDownstream := types.ResponseDownstream{
		Name:            d.Name,
		Links:           links,
		CurrentVersion:  appVersions.CurrentVersion,
		PendingVersions: appVersions.PendingVersions,
		PastVersions:    appVersions.PastVersions,
		GitOps:          responseGitOps,
		Cluster:         cluster,
	}

	responseApp := types.ResponseApp{
		ID:                             a.ID,
		Slug:                           a.Slug,
		Name:                           a.Name,
		Namespace:                      util.PodNamespace,
		IsAirgap:                       a.IsAirgap,
		CurrentSequence:                latestVersion.ParentSequence,
		UpstreamURI:                    a.UpstreamURI,
		IconURI:                        a.IconURI,
		CreatedAt:                      a.CreatedAt,
		UpdatedAt:                      a.UpdatedAt,
		LastUpdateCheckAt:              a.LastUpdateCheckAt,
		HasPreflight:                   a.HasPreflight,
		IsConfigurable:                 a.IsConfigurable,
		UpdateCheckerSpec:              a.UpdateCheckerSpec,
		AutoDeploy:                     a.AutoDeploy,
		AppState:                       appState,
		IsGitOpsSupported:              isGitopsSupported,
		IsIdentityServiceSupported:     license.Spec.IsIdentityServiceSupported,
		IsAppIdentityServiceSupported:  isAppIdentityServiceSupported,
		IsGeoaxisSupported:             license.Spec.IsGeoaxisSupported,
		IsSemverRequired:               license.Spec.IsSemverRequired,
		IsSupportBundleUploadSupported: license.Spec.IsSupportBundleUploadSupported,
		AllowRollback:                  allowRollback,
		AllowSnapshots:                 allowSnapshots,
		TargetKotsVersion:              targetKotsVersion,
		LicenseType:                    license.Spec.LicenseType,
		Downstream:                     responseDownstream,
	}

	return &responseApp, nil
}

type GetAppVersionHistoryResponse struct {
	downstreamtypes.DownstreamVersionHistory `json:",inline"`
}

func (h *Handler) GetAppVersionHistory(w http.ResponseWriter, r *http.Request) {
	pageSize := 20
	currentPage := 0
	pinLatest, _ := strconv.ParseBool(r.URL.Query().Get("pinLatest"))
	pinLatestDeployable, _ := strconv.ParseBool(r.URL.Query().Get("pinLatestDeployable"))

	if val := r.URL.Query().Get("pageSize"); val != "" {
		ps, err := strconv.Atoi(val)
		if err != nil {
			err = errors.Wrap(err, "failed to parse page size")
			logger.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		pageSize = ps
	}
	if val := r.URL.Query().Get("currentPage"); val != "" {
		cp, err := strconv.Atoi(val)
		if err != nil {
			err = errors.Wrap(err, "failed to parse current page")
			logger.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		currentPage = cp
	}

	appSlug := mux.Vars(r)["appSlug"]
	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		err = errors.Wrap(err, "failed to get app from slug")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(foundApp.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to list downstreams for app")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if len(downstreams) == 0 {
		err = errors.New("no downstreams for app")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clusterID := downstreams[0].ClusterID

	history, err := store.GetStore().GetDownstreamVersionHistory(foundApp.ID, clusterID, currentPage, pageSize, pinLatest, pinLatestDeployable)
	if err != nil {
		err = errors.Wrap(err, "failed to get downstream versions")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := GetAppVersionHistoryResponse{
		DownstreamVersionHistory: *history,
	}

	JSON(w, http.StatusOK, response)
}

type RemoveAppRequest struct {
	Undeploy bool `json:"undeploy"`
	Force    bool `json:"force"`
}

type RemoveAppResponse struct {
	Error string `json:"error,omitempty"`
}

func (h *Handler) RemoveApp(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	response := RemoveAppResponse{}

	removeAppRequest := RemoveAppRequest{}
	if err := json.NewDecoder(r.Body).Decode(&removeAppRequest); err != nil {
		response.Error = "failed to parse request body"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	app, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		if store.GetStore().IsNotFound(err) {
			response.Error = "app slug not found"
			logger.Error(errors.Wrap(err, response.Error))
			JSON(w, http.StatusNotFound, response)
		} else {
			response.Error = "failed to find app slug"
			logger.Error(errors.Wrap(err, response.Error))
			JSON(w, http.StatusInternalServerError, response)
		}
		return
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(app.ID)
	if err != nil {
		response.Error = "failed to list downstreams"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if len(downstreams) == 0 {
		response.Error = "no downstreams found for app"
		logger.Error(errors.New(response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}
	d := downstreams[0]

	if !removeAppRequest.Force && !removeAppRequest.Undeploy {
		currentVersion, err := store.GetStore().GetCurrentDownstreamVersion(app.ID, d.ClusterID)
		if err != nil {
			response.Error = "failed to get current downstream version"
			logger.Error(errors.Wrap(err, response.Error))
			JSON(w, http.StatusInternalServerError, response)
			return
		}

		if currentVersion != nil {
			response.Error = fmt.Sprintf("application %s is deployed and cannot be removed", appSlug)
			logger.Error(errors.Wrap(err, response.Error))
			JSON(w, http.StatusBadRequest, response)
			return
		}
	}

	if removeAppRequest.Undeploy {
		if err := operator.MustGetOperator().UndeployApp(app, &d, false); err != nil {
			response.Error = "failed to undeploy app"
			logger.Error(errors.Wrap(err, response.Error))
			JSON(w, http.StatusInternalServerError, response)
			return
		}
	}

	if err := store.GetStore().RemoveApp(app.ID); err != nil {
		response.Error = "failed to remove app"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	JSON(w, http.StatusOK, response)
}

type CanInstallAppVersionRequest struct {
	AppSpec    string `json:"appSpec"`
	AirgapSpec string `json:"airgapSpec"`
	IsInstall  bool   `json:"isInstall"`
}

type CanInstallAppVersionResponse struct {
	CanInstall bool   `json:"canInstall"`
	Error      string `json:"error,omitempty"`
}

func (h *Handler) CanInstallAppVersion(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	response := CanInstallAppVersionResponse{
		CanInstall: false,
	}

	request := CanInstallAppVersionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error = "failed to parse request body"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	if request.AppSpec != "" {
		response.CanInstall = false

		kotsApp, err := kotsutil.LoadKotsAppFromContents([]byte(request.AppSpec))
		if err != nil {
			response.Error = "failed to load kots app from contents"
			logger.Error(errors.Wrap(err, response.Error))
			JSON(w, http.StatusInternalServerError, response)
			return
		}

		if kotsApp != nil {
			response.CanInstall = kotsutil.IsKotsVersionCompatibleWithApp(*kotsApp, request.IsInstall)
		}

		if !response.CanInstall {
			response.Error = kotsutil.GetIncompatbileKotsVersionMessage(*kotsApp, request.IsInstall)
			JSON(w, http.StatusOK, response)
			return
		}
	}

	if request.AirgapSpec != "" && !request.IsInstall { // any version can be installed initially
		response.CanInstall = false

		a, err := store.GetStore().GetAppFromSlug(appSlug)
		if err != nil {
			response.Error = "failed to get kots app"
			logger.Error(errors.Wrap(err, response.Error))
			JSON(w, http.StatusInternalServerError, response)
			return
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		decoded, gvk, err := decode([]byte(request.AirgapSpec), nil, nil)
		if err != nil {
			response.Error = "failed to decode airgap spec"
			logger.Error(errors.Wrap(err, response.Error))
			JSON(w, http.StatusInternalServerError, response)
			return
		}

		if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Airgap" {
			response.Error = fmt.Sprintf("invalid airgap spec gvk: %s", gvk.String())
			logger.Error(errors.Wrap(err, response.Error))
			JSON(w, http.StatusInternalServerError, response)
			return
		}

		missingPrereqs, err := airgap.GetMissingRequiredVersions(a, decoded.(*kotsv1beta1.Airgap))
		if err != nil {
			response.Error = "failed to get release prerequisites"
			logger.Error(errors.Wrap(err, response.Error))
			JSON(w, http.StatusInternalServerError, response)
			return
		}

		if len(missingPrereqs) > 0 {
			response.Error = fmt.Sprintf("This airgap bundle cannot be uploaded because versions %s are required and must be uploaded first.", strings.Join(missingPrereqs, ", "))
			JSON(w, http.StatusOK, response)
			return
		}
	}

	// if we get here, everything passes
	response.CanInstall = true
	JSON(w, http.StatusOK, response)
}

type GetLatestDeployableVersionResponse struct {
	LatestDeployableVersion *downstreamtypes.DownstreamVersion `json:"latestDeployableVersion"`
	NumOfSkippedVersions    int                                `json:"numOfSkippedVersions"`
	NumOfRemainingVersions  int                                `json:"numOfRemainingVersions"`
	Error                   string                             `json:"error"`
}

func (h *Handler) GetLatestDeployableVersion(w http.ResponseWriter, r *http.Request) {
	getLatestDeployableVersionResponse := GetLatestDeployableVersionResponse{}

	appSlug := mux.Vars(r)["appSlug"]
	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		errMsg := "failed to get app from slug"
		logger.Error(errors.Wrap(err, errMsg))
		getLatestDeployableVersionResponse.Error = errMsg
		JSON(w, http.StatusBadRequest, getLatestDeployableVersionResponse)
		return
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		errMsg := "failed to list downstreams for app"
		logger.Error(errors.Wrap(err, errMsg))
		getLatestDeployableVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, getLatestDeployableVersionResponse)
		return
	} else if len(downstreams) == 0 {
		errMsg := "no downstreams for app"
		logger.Error(errors.New(errMsg))
		getLatestDeployableVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, getLatestDeployableVersionResponse)
		return
	}
	clusterID := downstreams[0].ClusterID

	latestDeployableVersion, numOfSkippedVersions, numOfRemainingVersions, err := store.GetStore().GetLatestDeployableDownstreamVersion(a.ID, clusterID)
	if err != nil {
		errMsg := "failed to get next downtream version"
		logger.Error(errors.Wrap(err, errMsg))
		getLatestDeployableVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, getLatestDeployableVersionResponse)
		return
	}

	getLatestDeployableVersionResponse.LatestDeployableVersion = latestDeployableVersion
	getLatestDeployableVersionResponse.NumOfSkippedVersions = numOfSkippedVersions
	getLatestDeployableVersionResponse.NumOfRemainingVersions = numOfRemainingVersions

	JSON(w, http.StatusOK, getLatestDeployableVersionResponse)
}

func (h *Handler) GetAutomatedInstallStatus(w http.ResponseWriter, r *http.Request) {
	status, msg, err := store.GetStore().GetTaskStatus(fmt.Sprintf("automated-install-slug-%s", mux.Vars(r)["appSlug"]))
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get install status for app %s", mux.Vars(r)["appSlug"]))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	response := kotsstore.TaskStatus{
		Status:  status,
		Message: msg,
	}

	JSON(w, http.StatusOK, response)
}
