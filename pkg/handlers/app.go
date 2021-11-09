package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/api/handlers/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/gitops"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/rbac"
	"github.com/replicatedhq/kots/pkg/registry"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/session"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/version"
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

	// Carefully now, peek at registry credentials to see if we need to prompt for them
	hasKurlRegistry, err := registry.HasKurlRegistry()
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to check registry status for pending app %s", papp.Slug))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pendingAppResponse := types.GetPendingAppResponse{
		App: types.ResponsePendingApp{
			ID:            papp.ID,
			Slug:          papp.Slug,
			Name:          papp.Name,
			LicenseData:   papp.LicenseData,
			NeedsRegistry: !hasKurlRegistry,
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
		parentSequence, err := store.GetStore().GetCurrentParentSequence(a.ID, d.ClusterID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current parent sequence for downstream")
		}

		links, err := version.GetRealizedLinksFromAppSpec(a.ID, parentSequence)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get realized links from app spec")
		}

		currentVersion, err := store.GetStore().GetCurrentVersion(a.ID, d.ClusterID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current downstream version")
		}

		pendingVersions, err := store.GetStore().GetPendingVersions(a.ID, d.ClusterID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get pending versions")
		}

		pastVersions, err := store.GetStore().GetPastVersions(a.ID, d.ClusterID)
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
		parentSequence, err := store.GetStore().GetCurrentParentSequence(a.ID, downstreams[0].ClusterID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current parent sequence for downstream")
		}

		s, err := store.GetStore().IsSnapshotsSupportedForVersion(a, parentSequence, &render.Renderer{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if snapshots is allowed")
		}
		allowSnapshots = s && license.Spec.IsSnapshotSupported
	}

	responseApp := types.ResponseApp{
		ID:                             a.ID,
		Slug:                           a.Slug,
		Name:                           a.Name,
		IsAirgap:                       a.IsAirgap,
		CurrentSequence:                a.CurrentSequence,
		UpstreamURI:                    a.UpstreamURI,
		IconURI:                        a.IconURI,
		CreatedAt:                      a.CreatedAt,
		UpdatedAt:                      a.UpdatedAt,
		LastUpdateCheckAt:              a.LastUpdateCheckAt,
		HasPreflight:                   a.HasPreflight,
		IsConfigurable:                 a.IsConfigurable,
		UpdateCheckerSpec:              a.UpdateCheckerSpec,
		IsGitOpsSupported:              license.Spec.IsGitOpsSupported,
		IsIdentityServiceSupported:     license.Spec.IsIdentityServiceSupported,
		IsAppIdentityServiceSupported:  isAppIdentityServiceSupported,
		IsGeoaxisSupported:             license.Spec.IsGeoaxisSupported,
		IsSupportBundleUploadSupported: license.Spec.IsSupportBundleUploadSupported,
		AllowRollback:                  allowRollback,
		AllowSnapshots:                 allowSnapshots,
		LicenseType:                    license.Spec.LicenseType,
		CurrentVersion:                 currentVersion,
		Downstreams:                    responseDownstreams,
	}

	return &responseApp, nil
}

type GetAppVersionsResponse struct {
	VersionHistory downstreamtypes.DownstreamVersions `json:"versionHistory"`
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

	currentVersion, err := store.GetStore().GetCurrentVersion(foundApp.ID, clusterID)
	if err != nil {
		err = errors.Wrap(err, "failed to get current downstream version")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pendingVersions, err := store.GetStore().GetPendingVersions(foundApp.ID, clusterID)
	if err != nil {
		err = errors.Wrap(err, "failed to get pending versions")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pastVersions, err := store.GetStore().GetPastVersions(foundApp.ID, clusterID)
	if err != nil {
		err = errors.Wrap(err, "failed to get past versions")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := GetAppVersionsResponse{
		VersionHistory: downstreamtypes.DownstreamVersions{},
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
			currentVersion, err := store.GetStore().GetCurrentVersion(app.ID, d.ClusterID)
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
