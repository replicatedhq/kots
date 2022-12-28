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
}

type DownstreamInfo struct {
	Cursor             string `json:"cursor,omitempty"`
	ChannelID          string `json:"channel_id,omitempty"`
	ChannelName        string `json:"channel_name,omitempty"`
	Sequence           *int64 `json:"sequence,omitempty"`
	Source             string `json:"source,omitempty"`
	Status             string `json:"status,omitempty"`
	PreflightState     string `json:"preflight_state,omitempty"`
	SkipPreflights     bool   `json:"skip_preflights,omitempty"`
	ReplHelmInstalls   int    `json:"repl_helm_installs,omitempty"`
	NativeHelmInstalls int    `json:"native_helm_installs,omitempty"`
}
