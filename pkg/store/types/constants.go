package types

type DownstreamVersionStatus string

const (
	VersionUnknown          DownstreamVersionStatus = "unknown"
	VersionPendingConfig    DownstreamVersionStatus = "pending_config"
	VersionPending          DownstreamVersionStatus = "pending"
	VersionPendingPreflight DownstreamVersionStatus = "pending_preflight"
	VersionPendingDownload  DownstreamVersionStatus = "pending_download"
	VersionDeploying        DownstreamVersionStatus = "deploying"
	VersionDeployed         DownstreamVersionStatus = "deployed"
	VersionFailed           DownstreamVersionStatus = "failed"
)
