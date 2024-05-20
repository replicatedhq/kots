package types

import "time"

type CheckForUpdatesOpts struct {
	AppID                  string
	DeployLatest           bool
	DeployVersionLabel     string
	IsAutomatic            bool
	SkipPreflights         bool
	SkipCompatibilityCheck bool
	IsCLI                  bool
	Wait                   bool
}

type UpdateCheckResponse struct {
	AvailableUpdates  int64
	CurrentRelease    *UpdateCheckRelease
	AvailableReleases []UpdateCheckRelease
	DeployingRelease  *UpdateCheckRelease
}

type UpdateCheckRelease struct {
	Sequence int64
	Version  string
}

type AvailableUpdate struct {
	VersionLabel       string     `json:"versionLabel"`
	UpdateCursor       string     `json:"updateCursor"`
	ChannelID          string     `json:"channelId"`
	IsRequired         bool       `json:"isRequired"`
	UpstreamReleasedAt *time.Time `json:"upstreamReleasedAt,omitempty"`
	ReleaseNotes       string     `json:"releaseNotes,omitempty"`
	IsDeployable       bool       `json:"isDeployable,omitempty"`
	NonDeployableCause string     `json:"nonDeployableCause,omitempty"`
}
