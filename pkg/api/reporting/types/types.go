package types

type ReportingInfo struct {
	InstanceID         string         `json:"instance_id"`
	ClusterID          string         `json:"cluster_id"`
	Downstream         DownstreamInfo `json:"downstream"`
	AppStatus          string         `json:"app_status"`
	IsKurl             bool           `json:"is_kurl"`
	KurlNodeCountTotal int            `json:"kurl_node_count_total"`
	KurlNodeCountReady int            `json:"kurl_node_count_ready"`
	K8sVersion         string         `json:"k8s_version"`
	KOTSInstallID      string         `json:"kots_install_id"`
	KURLInstallID      string         `json:"kurl_install_id"`
	IsGitOpsEnabled    bool           `json:"is_gitops_enabled"`
	GitOpsProvider     string         `json:"gitops_provider"`
}

type GitOpsReport struct {
	IsEnabled bool   `json:"is_enabled"`
	Provider  string `json:"provider"`
}

type DownstreamInfo struct {
	Cursor             string `json:"cursor"`
	ChannelID          string `json:"channel_id"`
	ChannelName        string `json:"channel_name"`
	Sequence           *int64 `json:"sequence"`
	Source             string `json:"source"`
	Status             string `json:"status"`
	PreflightState     string `json:"preflight_state"`
	SkipPreflights     bool   `json:"skip_preflights"`
	ReplHelmInstalls   int    `json:"repl_helm_installs"`
	NativeHelmInstalls int    `json:"native_helm_installs"`
}
