package types

import (
	"path"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
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
	EncryptionKey string
}

type WriteOptions struct {
	RootDir             string
	CreateAppDir        bool
	IncludeAdminConsole bool
	HTTPProxyEnvValue   string
	HTTPSProxyEnvValue  string
	NoProxyEnvValue     string
	// This should be set to true when updating due to license sync, config update, registry settings update.
	// and should be false when it's an upstream update.
	// When true, the channel name in Installation yaml will not be changed.
	PreserveInstallation bool
	// Set to true on initial installation when an unencrypted config file is provided
	EncryptConfig  bool
	SharedPassword string
}

type FetchOptions struct {
	RootDir             string
	UseAppDir           bool
	HelmRepoName        string
	HelmRepoURI         string
	HelmOptions         []string
	LocalPath           string
	License             *kotsv1beta1.License
	ConfigValues        *kotsv1beta1.ConfigValues
	Airgap              *kotsv1beta1.Airgap
	EncryptionKey       string
	CurrentCursor       string
	CurrentChannelID    string
	CurrentChannelName  string
	CurrentVersionLabel string
	AppSequence         int64
	LocalRegistry       LocalRegistry
	ReportingInfo       *ReportingInfo
}

type LocalRegistry struct {
	Host      string
	Namespace string
	Username  string
	Password  string
}

type ReportingInfo struct {
	AppID                 string
	ClusterID             string
	DownstreamCursor      string
	DownstreamChannelID   string
	DownstreamChannelName string
	AppStatus             string
	IsKurl                bool
	K8sVersion            string
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
