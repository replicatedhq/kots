package types

import "time"

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
