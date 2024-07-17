package types

type ReportingInfo struct {
	InstanceID              string         `json:"instance_id" yaml:"instance_id"`
	ClusterID               string         `json:"cluster_id" yaml:"cluster_id"`
	Downstream              DownstreamInfo `json:"downstream" yaml:"downstream"`
	AppStatus               string         `json:"app_status" yaml:"app_status"`
	IsKurl                  bool           `json:"is_kurl" yaml:"is_kurl"`
	KurlNodeCountTotal      int            `json:"kurl_node_count_total" yaml:"kurl_node_count_total"`
	KurlNodeCountReady      int            `json:"kurl_node_count_ready" yaml:"kurl_node_count_ready"`
	K8sVersion              string         `json:"k8s_version" yaml:"k8s_version"`
	K8sDistribution         string         `json:"k8s_distribution" yaml:"k8s_distribution"`
	UserAgent               string         `json:"user_agent" yaml:"user_agent"`
	KOTSInstallID           string         `json:"kots_install_id" yaml:"kots_install_id"`
	KURLInstallID           string         `json:"kurl_install_id" yaml:"kurl_install_id"`
	EmbeddedClusterID       string         `json:"embedded_cluster_id" yaml:"embedded_cluster_id"`
	EmbeddedClusterVersion  string         `json:"embedded_cluster_version" yaml:"embedded_cluster_version"`
	IsGitOpsEnabled         bool           `json:"is_gitops_enabled" yaml:"is_gitops_enabled"`
	GitOpsProvider          string         `json:"gitops_provider" yaml:"gitops_provider"`
	SnapshotProvider        string         `json:"snapshot_provider" yaml:"snapshot_provider"`
	SnapshotFullSchedule    string         `json:"snapshot_full_schedule" yaml:"snapshot_full_schedule"`
	SnapshotFullTTL         string         `json:"snapshot_full_ttl" yaml:"snapshot_full_ttl"`
	SnapshotPartialSchedule string         `json:"snapshot_partial_schedule" yaml:"snapshot_partial_schedule"`
	SnapshotPartialTTL      string         `json:"snapshot_partial_ttl" yaml:"snapshot_partial_ttl"`
}

type DownstreamInfo struct {
	Cursor             string `json:"cursor" yaml:"cursor"`
	ChannelID          string `json:"channel_id" yaml:"channel_id"`
	ChannelName        string `json:"channel_name" yaml:"channel_name"`
	Sequence           *int64 `json:"sequence" yaml:"sequence"`
	Source             string `json:"source" yaml:"source"`
	Status             string `json:"status" yaml:"status"`
	PreflightState     string `json:"preflight_state" yaml:"preflight_state"`
	SkipPreflights     bool   `json:"skip_preflights" yaml:"skip_preflights"`
	ReplHelmInstalls   int    `json:"repl_helm_installs" yaml:"repl_helm_installs"`
	NativeHelmInstalls int    `json:"native_helm_installs" yaml:"native_helm_installs"`
}
