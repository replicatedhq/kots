package types

import (
	"path"
	"time"

	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

type UpstreamFile struct {
	Path    string
	Content []byte
}

type Upstream struct {
	URI                      string
	Name                     string
	Type                     string
	Files                    []UpstreamFile
	UpdateCursor             string
	License                  *kotsv1beta1.License
	ChannelID                string
	ChannelName              string
	VersionLabel             string
	IsRequired               bool
	ReleaseNotes             string
	ReleasedAt               *time.Time
	ReplicatedRegistryDomain string
	ReplicatedProxyDomain    string
	ReplicatedChartNames     []string
	EmbeddedClusterArtifacts *kotsv1beta1.EmbeddedClusterArtifacts
	EncryptionKey            string
}

type Update struct {
	ChannelID    string     `json:"channelID"`
	ChannelName  string     `json:"channelName"`
	Cursor       string     `json:"cursor"`
	VersionLabel string     `json:"versionLabel"`
	IsRequired   bool       `json:"isRequired"`
	ReleaseNotes string     `json:"releaseNotes"`
	ReleasedAt   *time.Time `json:"releasedAt"`
	AppSequence  *int64     `json:"appSequence"` // can have a sequence if update is available as a pending download app version
}

type UpdateCheckResult struct {
	UpdateCheckTime time.Time `json:"updateCheckTime"`
	Updates         []Update  `json:"updates"`
}

type WriteOptions struct {
	RootDir              string
	Namespace            string
	CreateAppDir         bool
	IncludeAdminConsole  bool
	IncludeMinio         bool
	MigrateToMinioXl     bool
	CurrentMinioImage    string
	StorageClassName     string
	HTTPProxyEnvValue    string
	HTTPSProxyEnvValue   string
	NoProxyEnvValue      string
	IsMinimalRBAC        bool
	AdditionalNamespaces []string
	IsAirgap             bool
	KotsadmID            string
	AppID                string
	// This should be set to true when updating due to license sync, config update, registry settings update.
	// and should be false when it's an upstream update.
	// When true, the channel name in Installation yaml will not be changed.
	PreserveInstallation bool
	// Set to true on initial installation when an unencrypted config file is provided
	EncryptConfig  bool
	SharedPassword string
	IsOpenShift    bool
	IsGKEAutopilot bool

	RegistryConfig kotsadmtypes.RegistryConfig
}

type FetchOptions struct {
	RootDir                         string
	UseAppDir                       bool
	HelmRepoURI                     string
	LocalPath                       string
	License                         *kotsv1beta1.License
	ConfigValues                    *kotsv1beta1.ConfigValues
	IdentityConfig                  *kotsv1beta1.IdentityConfig
	Airgap                          *kotsv1beta1.Airgap
	EncryptionKey                   string
	LastUpdateCheckAt               *time.Time
	CurrentCursor                   string
	CurrentChannelID                string
	CurrentChannelName              string
	CurrentVersionLabel             string
	CurrentVersionIsRequired        bool
	CurrentReplicatedRegistryDomain string
	CurrentReplicatedProxyDomain    string
	CurrentReplicatedChartNames     []string
	CurrentEmbeddedClusterArtifacts *kotsv1beta1.EmbeddedClusterArtifacts
	ChannelChanged                  bool
	SortOrder                       string
	AppSlug                         string
	AppSequence                     int64
	AppVersionLabel                 string
	LocalRegistry                   registrytypes.RegistrySettings
	ReportingInfo                   *reportingtypes.ReportingInfo
	SkipCompatibilityCheck          bool
}

func (u *Upstream) GetUpstreamDir(options WriteOptions) string {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	return path.Join(renderDir, "upstream")
}

func (u *Upstream) GetBaseDir(options WriteOptions) string {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	return path.Join(renderDir, "base")
}

func (u *Upstream) GetOverlaysDir(options WriteOptions) string {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	return path.Join(renderDir, "overlays")
}

func (u *Upstream) GetRenderedDir(options WriteOptions) string {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	return path.Join(renderDir, "rendered")
}

func (u *Upstream) GetKotsKindsDir(options WriteOptions) string {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	return path.Join(renderDir, "kotsKinds")
}

func (u *Upstream) GetHelmDir(options WriteOptions) string {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	return path.Join(renderDir, "helm")
}

func (u *Upstream) GetSkippedDir(options WriteOptions) string {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	return path.Join(renderDir, "skippedFiles")
}
