package handlers

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/gitops"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	versiontypes "github.com/replicatedhq/kots/kotsadm/pkg/version/types"
)

type GetAppResponse struct {
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
	BundleCommand     string     `json:"bundleCommand"`
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
	a, err := app.GetFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	isGitOpsSupported, err := version.IsGitOpsSupported(a.ID, a.CurrentSequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	allowRollback, err := version.IsAllowRollback(a.ID, a.CurrentSequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	licenseType, err := version.GetLicenseType(a.ID, a.CurrentSequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	currentVersion, err := version.Get(a.ID, a.CurrentSequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	downstreams, err := downstream.ListDownstreamsForApp(a.ID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responseDownstreams := []ResponseDownstream{}
	for _, d := range downstreams {
		parentSequence, err := downstream.GetCurrentParentSequence(a.ID, d.ClusterID)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		links, err := version.GetRealizedLinksFromAppSpec(a.ID, parentSequence)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		currentVersion, err := downstream.GetCurrentVersion(a.ID, d.ClusterID)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		pendingVersions, err := downstream.GetPendingVersions(a.ID, d.ClusterID)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		pastVersions, err := downstream.GetPastVersions(a.ID, d.ClusterID)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		downstreamGitOps, err := gitops.GetDownstreamGitOps(a.ID, d.ClusterID)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
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
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		s, err := version.IsAllowSnapshots(a.ID, parentSequence)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		allowSnapshots = s
	}

	getAppResponse := GetAppResponse{
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
		BundleCommand:     a.BundleCommand,
		HasPreflight:      a.HasPreflight,
		IsConfigurable:    a.IsConfigurable,
		UpdateCheckerSpec: a.UpdateCheckerSpec,
		IsGitOpsSupported: isGitOpsSupported,
		AllowRollback:     allowRollback,
		AllowSnapshots:    allowSnapshots,
		LicenseType:       licenseType,
		CurrentVersion:    currentVersion,
		Downstreams:       responseDownstreams,
	}

	JSON(w, http.StatusOK, getAppResponse)
}
