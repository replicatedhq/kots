package types

import (
	"path"
	"time"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
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
	URI           string
	Name          string
	Type          string
	Files         []UpstreamFile
	UpdateCursor  string
	ChannelID     string
	ChannelName   string
	VersionLabel  string
	ReleaseNotes  string
	ReleasedAt    *time.Time
	EncryptionKey string
}

type Update struct {
	ChannelID    string     `json:"channelID"`
	ChannelName  string     `json:"channelName"`
	Cursor       string     `json:"cursor"`
	VersionLabel string     `json:"versionLabel"`
	ReleaseNotes string     `json:"releaseNotes"`
	ReleasedAt   *time.Time `json:"releasedAt"`
	AppSequence  *int64     `json:"appSequence"` // can have a sequence if update is available as a pending download app version
}

type WriteOptions struct {
	RootDir              string
	Namespace            string
	CreateAppDir         bool
	IncludeAdminConsole  bool
	IncludeMinio         bool
	HTTPProxyEnvValue    string
	HTTPSProxyEnvValue   string
	NoProxyEnvValue      string
	IsMinimalRBAC        bool
	AdditionalNamespaces []string
	// This should be set to true when updating due to license sync, config update, registry settings update.
	// and should be false when it's an upstream update.
	// When true, the channel name in Installation yaml will not be changed.
	PreserveInstallation bool
	// Set to true on initial installation when an unencrypted config file is provided
	EncryptConfig  bool
	SharedPassword string
	IsOpenShift    bool
}

type FetchOptions struct {
	RootDir                string
	UseAppDir              bool
	HelmRepoName           string
	HelmRepoURI            string
	HelmOptions            []string
	LocalPath              string
	License                *kotsv1beta1.License
	ConfigValues           *kotsv1beta1.ConfigValues
	IdentityConfig         *kotsv1beta1.IdentityConfig
	Airgap                 *kotsv1beta1.Airgap
	EncryptionKey          string
	LastUpdateCheckAt      *time.Time
	CurrentCursor          string
	CurrentChannelID       string
	CurrentChannelName     string
	CurrentVersionLabel    string
	ChannelChanged         bool
	AppSlug                string
	AppSequence            int64
	AppVersionLabel        string
	LocalRegistry          LocalRegistry
	ReportingInfo          *reportingtypes.ReportingInfo
	IdentityPostgresConfig *kotsv1beta1.IdentityPostgresConfig
	SkipCompatibilityCheck bool
}

type LocalRegistry struct {
	Host      string
	Namespace string
	Username  string
	Password  string
	ReadOnly  bool
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

func (u *Upstream) GetSkippedDir(options WriteOptions) string {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	return path.Join(renderDir, "skippedFiles")
}
