package types

import "time"

type UndeployStatus string

const (
	UndeployInProcess UndeployStatus = "in_process"
	UndeployCompleted UndeployStatus = "completed"
	UndeployFailed    UndeployStatus = "failed"
	UndeployReset     UndeployStatus = ""
)

type AutoDeploy string

const (
	AutoDeployDisabled              AutoDeploy = "disabled"
	AutoDeploySemverPatch           AutoDeploy = "semver-patch"
	AutoDeploySemverMinorPatch      AutoDeploy = "semver-minor-patch"
	AutoDeploySemverMajorMinorPatch AutoDeploy = "semver-major-minor-patch"
	AutoDeploySequence              AutoDeploy = "sequence"
)

type App struct {
	ID                    string         `json:"id"`
	Slug                  string         `json:"slug"`
	Name                  string         `json:"name"`
	License               string         `json:"license"`
	IsAirgap              bool           `json:"isAirgap"`
	CurrentSequence       int64          `json:"currentSequence"`
	UpstreamURI           string         `json:"upstreamUri"`
	IconURI               string         `json:"iconUri"`
	UpdatedAt             *time.Time     `json:"createdAt"`
	CreatedAt             time.Time      `json:"updatedAt"`
	LastUpdateCheckAt     *time.Time     `json:"lastUpdateCheckAt"`
	HasPreflight          bool           `json:"hasPreflight"`
	IsConfigurable        bool           `json:"isConfigurable"`
	SnapshotTTL           string         `json:"snapshotTtl"`
	SnapshotSchedule      string         `json:"snapshotSchedule"`
	RestoreInProgressName string         `json:"restoreInProgressName"`
	RestoreUndeployStatus UndeployStatus `json:"restoreUndeloyStatus"`
	UpdateCheckerSpec     string         `json:"updateCheckerSpec"`
	AutoDeploy            AutoDeploy     `json:"autoDeploy"`
	IsGitOps              bool           `json:"isGitOps"`
	InstallState          string         `json:"installState"`
	LastLicenseSync       string         `json:"lastLicenseSync"`
	ChannelChanged        bool           `json:"channelChanged"`
}
