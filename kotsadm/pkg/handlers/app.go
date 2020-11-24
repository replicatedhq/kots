package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/gitops"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	versiontypes "github.com/replicatedhq/kots/kotsadm/pkg/version/types"
	"github.com/replicatedhq/kots/pkg/rbac"
)

type ListAppsResponse struct {
	Apps []ResponseApp `json:"apps"`
}

type ResponseApp struct {
	ID                string     `json:"id"`
	Slug              string     `json:"slug"`
	Name              string     `json:"name"`
	IsAirgap          bool       `json:"isAirgap"`
	CurrentSequence   int64      `json:"currentSequence"`
	UpstreamURI       string     `json:"upstreamUri"`
	IconURI           string     `json:"iconUri"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         *time.Time `json:"updatedAt"`
	LastUpdateCheckAt string     `json:"lastUpdateCheckAt"`
	BundleCommand     []string   `json:"bundleCommand"`
	HasPreflight      bool       `json:"hasPreflight"`
	IsConfigurable    bool       `json:"isConfigurable"`
	UpdateCheckerSpec string     `json:"updateCheckerSpec"`

	IsGitOpsSupported bool                     `json:"isGitOpsSupported"`
	AllowRollback     bool                     `json:"allowRollback"`
	AllowSnapshots    bool                     `json:"allowSnapshots"`
	LicenseType       string                   `json:"licenseType"`
	CurrentVersion    *versiontypes.AppVersion `json:"currentVersion"`

	Downstreams []ResponseDownstream `json:"downstreams"`
}

type ResponseDownstream struct {
	Name            string                              `json:"name"`
	Links           []versiontypes.RealizedLink         `json:"links"`
	CurrentVersion  *downstreamtypes.DownstreamVersion  `json:"currentVersion"`
	PendingVersions []downstreamtypes.DownstreamVersion `json:"pendingVersions"`
	PastVersions    []downstreamtypes.DownstreamVersion `json:"pastVersions"`
	GitOps          ResponseGitOps                      `json:"gitops"`
	Cluster         ResponseCluster                     `json:"cluster"`
}

type ResponseGitOps struct {
	Enabled     bool   `json:"enabled"`
	Provider    string `json:"provider"`
	Uri         string `json:"uri"`
	Hostname    string `json:"hostname"`
	Path        string `json:"path"`
	Branch      string `json:"branch"`
	Format      string `json:"format"`
	Action      string `json:"action"`
	DeployKey   string `json:"deployKey"`
	IsConnected bool   `json:"isConnected"`
}

type ResponseCluster struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

func ListApps(w http.ResponseWriter, r *http.Request) {
	sess := GetSession(r)
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

	appSlugs := []string{}
	for _, a := range apps {
		appSlugs = append(appSlugs, a.Slug)
	}

	responseApps := []ResponseApp{}
	for _, a := range apps {
		allow, err := rbac.CheckAccess(r.Context(), "read", fmt.Sprintf("app.%s", a.Slug), sess.Roles, appSlugs)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to check access for app %s", a.Slug))
			w.WriteHeader(http.StatusInternalServerError)
			return
			return
		} else if !allow {
			continue
		}

		responseApp, err := responseAppFromApp(a)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		responseApps = append(responseApps, *responseApp)
	}

	listAppsResponse := ListAppsResponse{
		Apps: responseApps,
	}

	JSON(w, http.StatusOK, listAppsResponse)
}

func GetApp(w http.ResponseWriter, r *http.Request) {
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

func responseAppFromApp(a *apptypes.App) (*ResponseApp, error) {
	isGitOpsSupported, err := store.GetStore().IsGitOpsSupportedForVersion(a.ID, a.CurrentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if gitops is supported")
	}

	allowRollback, err := store.GetStore().IsRollbackSupportedForVersion(a.ID, a.CurrentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if rollback is supported")
	}

	license, err := store.GetStore().GetLicenseForAppVersion(a.ID, a.CurrentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get license")
	}

	currentVersion, err := store.GetStore().GetAppVersion(a.ID, a.CurrentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version")
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}

	responseDownstreams := []ResponseDownstream{}
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
		responseGitOps := ResponseGitOps{}
		if downstreamGitOps != nil {
			responseGitOps = ResponseGitOps{
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

		cluster := ResponseCluster{
			ID:   d.ClusterID,
			Slug: d.ClusterSlug,
		}

		responseDownstream := ResponseDownstream{
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
		allowSnapshots = s
	}

	responseApp := ResponseApp{
		ID:                a.ID,
		Slug:              a.Slug,
		Name:              a.Name,
		IsAirgap:          a.IsAirgap,
		CurrentSequence:   a.CurrentSequence,
		UpstreamURI:       a.UpstreamURI,
		IconURI:           a.IconURI,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
		LastUpdateCheckAt: a.LastUpdateCheckAt,
		BundleCommand:     supportbundle.GetBundleCommand(a.Slug),
		HasPreflight:      a.HasPreflight,
		IsConfigurable:    a.IsConfigurable,
		UpdateCheckerSpec: a.UpdateCheckerSpec,
		IsGitOpsSupported: isGitOpsSupported,
		AllowRollback:     allowRollback,
		AllowSnapshots:    allowSnapshots,
		LicenseType:       license.Spec.LicenseType,
		CurrentVersion:    currentVersion,
		Downstreams:       responseDownstreams,
	}

	return &responseApp, nil
}

type GetAppVersionsResponse struct {
	VersionHistory []downstreamtypes.DownstreamVersion `json:"versionHistory"`
}

func GetAppVersionHistory(w http.ResponseWriter, r *http.Request) {
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

func RemoveApp(w http.ResponseWriter, r *http.Request) {
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
