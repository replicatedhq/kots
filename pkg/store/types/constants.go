package types

type DownstreamVersionStatus string

const (
	VersionUnknown                  DownstreamVersionStatus = "unknown"                    // we don't know
	VersionPendingClusterManagement DownstreamVersionStatus = "pending_cluster_management" // needs cluster configuration
	VersionPendingConfig            DownstreamVersionStatus = "pending_config"             // needs required configuration
	VersionPendingDownload          DownstreamVersionStatus = "pending_download"           // needs to be downloaded from the upstream source
	VersionPendingPreflight         DownstreamVersionStatus = "pending_preflight"          // waiting for preflights to finish
	VersionPending                  DownstreamVersionStatus = "pending"                    // can be deployed, but is not yet
	VersionDeploying                DownstreamVersionStatus = "deploying"                  // is being deployed
	VersionDeployed                 DownstreamVersionStatus = "deployed"                   // did deploy successfully
	VersionFailed                   DownstreamVersionStatus = "failed"                     // did not deploy successfully
)
