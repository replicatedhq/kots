package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/gitops"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/api/handlers/types"
	"github.com/replicatedhq/kots/pkg/rbac"
)

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

	defaultRoles := rbac.DefaultRoles() // TODO (ethan): this should be set in the handler

	responseApps := []types.ResponseApp{}
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

	isIdentityServiceSupportedForVersion, err := store.GetStore().IsIdentityServiceSupportedForVersion(a.ID, a.CurrentSequence)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check if identity service is supported for version %d", a.CurrentSequence)
	}
	isAppIdentityServiceSupported := isIdentityServiceSupportedForVersion && license.Spec.IsIdentityServiceSupported

	allowRollback, err := store.GetStore().IsRollbackSupportedForVersion(a.ID, a.CurrentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if rollback is supported")
	}

	currentVersion, err := store.GetStore().GetAppVersion(a.ID, a.CurrentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version")
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}

	responseDownstreams := []types.ResponseDownstream{}
	for _, d := range downstreams {
		parentSequence, err := downstream.GetCurrentParentSequence(a.ID, d.ClusterID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current parent sequence for downstream")
		}

		links, err := version.GetRealizedLinksFromAppSpec(a.ID, parentSequence)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get realized links from app spec")
		}

		currentVersion, err := downstream.GetCurrentVersion(a.ID, d.ClusterID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current downstream version")
		}

		pendingVersions, err := downstream.GetPendingVersions(a.ID, d.ClusterID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get pending versions")
		}

		pastVersions, err := downstream.GetPastVersions(a.ID, d.ClusterID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get past versions")
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

		responseDownstream := types.ResponseDownstream{
			Name:            d.Name,
			Links:           links,
			CurrentVersion:  currentVersion,
			PendingVersions: pendingVersions,
			PastVersions:    pastVersions,
			GitOps:          responseGitOps,
			Cluster:         cluster,
		}

		responseDownstreams = append(responseDownstreams, responseDownstream)
	}

	// check snapshots for the parent sequence of the deployed version
	allowSnapshots := false
	if len(downstreams) > 0 {
		parentSequence, err := downstream.GetCurrentParentSequence(a.ID, downstreams[0].ClusterID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current parent sequence for downstream")
		}

		s, err := store.GetStore().IsSnapshotsSupportedForVersion(a, parentSequence)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if snapshots is allowed")
		}
		allowSnapshots = s && license.Spec.IsSnapshotSupported
	}

	responseApp := types.ResponseApp{
		ID:                            a.ID,
		Slug:                          a.Slug,
		Name:                          a.Name,
		IsAirgap:                      a.IsAirgap,
		CurrentSequence:               a.CurrentSequence,
		UpstreamURI:                   a.UpstreamURI,
		IconURI:                       a.IconURI,
		CreatedAt:                     a.CreatedAt,
		UpdatedAt:                     a.UpdatedAt,
		LastUpdateCheckAt:             a.LastUpdateCheckAt,
		BundleCommand:                 supportbundle.GetBundleCommand(a.Slug),
		HasPreflight:                  a.HasPreflight,
		IsConfigurable:                a.IsConfigurable,
		UpdateCheckerSpec:             a.UpdateCheckerSpec,
		IsGitOpsSupported:             license.Spec.IsGitOpsSupported,
		IsIdentityServiceSupported:    license.Spec.IsIdentityServiceSupported,
		IsAppIdentityServiceSupported: isAppIdentityServiceSupported,
		IsGeoaxisSupported:            license.Spec.IsGeoaxisSupported,
		AllowRollback:                 allowRollback,
		AllowSnapshots:                allowSnapshots,
		LicenseType:                   license.Spec.LicenseType,
		CurrentVersion:                currentVersion,
		Downstreams:                   responseDownstreams,
	}

	return &responseApp, nil
}

type GetAppVersionsResponse struct {
	VersionHistory []downstreamtypes.DownstreamVersion `json:"versionHistory"`
}

func (h *Handler) GetAppVersionHistory(w http.ResponseWriter, r *http.Request) {
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

	currentVersion, err := downstream.GetCurrentVersion(foundApp.ID, clusterID)
	if err != nil {
		err = errors.Wrap(err, "failed to get current downstream version")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pendingVersions, err := downstream.GetPendingVersions(foundApp.ID, clusterID)
	if err != nil {
		err = errors.Wrap(err, "failed to get pending versions")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pastVersions, err := downstream.GetPastVersions(foundApp.ID, clusterID)
	if err != nil {
		err = errors.Wrap(err, "failed to get past versions")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := GetAppVersionsResponse{
		VersionHistory: []downstreamtypes.DownstreamVersion{},
	}
	response.VersionHistory = append(response.VersionHistory, pendingVersions...)
	if currentVersion != nil {
		response.VersionHistory = append(response.VersionHistory, *currentVersion)
	}
	response.VersionHistory = append(response.VersionHistory, pastVersions...)

	JSON(w, http.StatusOK, response)
}

type RemoveAppRequest struct {
	Force bool `json:"force"`
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

	if !removeAppRequest.Force {
		downstreams, err := store.GetStore().ListDownstreamsForApp(app.ID)
		if err != nil {
			response.Error = "failed to list downstreams"
			logger.Error(errors.Wrap(err, response.Error))
			JSON(w, http.StatusInternalServerError, response)
			return
		}

		for _, d := range downstreams {
			currentVersion, err := downstream.GetCurrentVersion(app.ID, d.ClusterID)
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
	}

	err = store.GetStore().RemoveApp(app.ID)
	if err != nil {
		response.Error = "failed to remove app"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	JSON(w, http.StatusOK, response)
}
