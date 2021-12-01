package types

import (
	"time"

	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	versiontypes "github.com/replicatedhq/kots/pkg/api/version/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
)

type ListAppsResponse struct {
	Apps []ResponseApp `json:"apps"`
}

type AppStatusResponse struct {
	AppStatus *appstatetypes.AppStatus `json:"appstatus"`
}

type ResponseApp struct {
	ID                string                    `json:"id"`
	Slug              string                    `json:"slug"`
	Name              string                    `json:"name"`
	IsAirgap          bool                      `json:"isAirgap"`
	CurrentSequence   int64                     `json:"currentSequence"`
	UpstreamURI       string                    `json:"upstreamUri"`
	IconURI           string                    `json:"iconUri"`
	CreatedAt         time.Time                 `json:"createdAt"`
	UpdatedAt         *time.Time                `json:"updatedAt"`
	LastUpdateCheckAt string                    `json:"lastUpdateCheckAt"`
	HasPreflight      bool                      `json:"hasPreflight"`
	IsConfigurable    bool                      `json:"isConfigurable"`
	UpdateCheckerSpec string                    `json:"updateCheckerSpec"`
	SemverAutoDeploy  apptypes.SemverAutoDeploy `json:"semverAutoDeploy"`

	IsGitOpsSupported              bool                     `json:"isGitOpsSupported"`
	IsIdentityServiceSupported     bool                     `json:"isIdentityServiceSupported"`
	IsAppIdentityServiceSupported  bool                     `json:"isAppIdentityServiceSupported"`
	IsGeoaxisSupported             bool                     `json:"isGeoaxisSupported"`
	IsNewKOTSUIEnabled             bool                     `json:"isNewKotsUiEnabled"`
	IsSupportBundleUploadSupported bool                     `json:"isSupportBundleUploadSupported"`
	AllowRollback                  bool                     `json:"allowRollback"`
	AllowSnapshots                 bool                     `json:"allowSnapshots"`
	LicenseType                    string                   `json:"licenseType"`
	CurrentVersion                 *versiontypes.AppVersion `json:"currentVersion"`

	Downstreams []ResponseDownstream `json:"downstreams"`
}

type ResponseDownstream struct {
	Name            string                               `json:"name"`
	Links           []versiontypes.RealizedLink          `json:"links"`
	CurrentVersion  *downstreamtypes.DownstreamVersion   `json:"currentVersion"`
	PendingVersions []*downstreamtypes.DownstreamVersion `json:"pendingVersions"`
	PastVersions    []*downstreamtypes.DownstreamVersion `json:"pastVersions"`
	GitOps          ResponseGitOps                       `json:"gitops"`
	Cluster         ResponseCluster                      `json:"cluster"`
}

type ResponseGitOps struct {
	Enabled     bool   `json:"enabled"`
	Provider    string `json:"provider"`
	Uri         string `json:"uri"`
	Hostname    string `json:"hostname"`
	HTTPPort    string `json:"httpPort"`
	SSHPort     string `json:"sshPort"`
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

type GetPendingAppResponse struct {
	App ResponsePendingApp `json:"app"`
}

type ResponsePendingApp struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	LicenseData   string `json:"licenseData"`
	NeedsRegistry bool   `json:"needsRegistry"`
}
