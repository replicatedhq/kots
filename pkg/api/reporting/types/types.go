package types

// This type is mimicked in the instance_report table.
type ReportingInfo struct {
	InstanceID              string         `json:"instance_id"`
	ClusterID               string         `json:"cluster_id"`
	Downstream              DownstreamInfo `json:"downstream"`
	AppStatus               string         `json:"app_status"`
	IsKurl                  bool           `json:"is_kurl"`
	KurlNodeCountTotal      int            `json:"kurl_node_count_total"`
	KurlNodeCountReady      int            `json:"kurl_node_count_ready"`
	K8sVersion              string         `json:"k8s_version"`
	K8sDistribution         string         `json:"k8s_distribution"`
	UserAgent               string         `json:"user_agent"`
	KOTSInstallID           string         `json:"kots_install_id"`
	KURLInstallID           string         `json:"kurl_install_id"`
	IsGitOpsEnabled         bool           `json:"is_gitops_enabled"`
	GitOpsProvider          string         `json:"gitops_provider"`
	SnapshotProvider        string         `json:"snapshot_provider"`
	SnapshotFullSchedule    string         `json:"snapshot_full_schedule"`
	SnapshotFullTTL         string         `json:"snapshot_full_ttl"`
	SnapshotPartialSchedule string         `json:"snapshot_partial_schedule"`
	SnapshotPartialTTL      string         `json:"snapshot_partial_ttl"`
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

type SnapshotReport struct {
	Provider        string
	Schedule        string
	RetentionPolicy string
}

// This type is mimicked in the preflight_report table.
type PreflightStatus struct {
	InstanceID      string `json:"instance_id"`
	ClusterID       string `json:"cluster_id"`
	Sequence        int64  `json:"sequence"`
	SkipPreflights  bool   `json:"skip_preflights"`
	InstallStatus   string `json:"install_status"`
	IsCLI           bool   `json:"is_cli"`
	PreflightStatus string `json:"preflight_status"`
	AppStatus       string `json:"app_status"`
	KOTSVersion     string `json:"kots_version"`
}
