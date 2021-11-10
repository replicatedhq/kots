package types

import "time"

type UndeployStatus string

const (
	UndeployInProcess UndeployStatus = "in_process"
	UndeployCompleted UndeployStatus = "completed"
	UndeployFailed    UndeployStatus = "failed"
	UndeployReset     UndeployStatus = ""
)

type SemverAutoDeploy string

const (
	SemverAutoDeployDisabled        SemverAutoDeploy = "disabled"
	SemverAutoDeployPatch           SemverAutoDeploy = "patch"
	SemverAutoDeployMinorPatch      SemverAutoDeploy = "minor-patch"
	SemverAutoDeployMajorMinorPatch SemverAutoDeploy = "major-minor-patch"
)

type App struct {
	ID                    string           `json:"id"`
	Slug                  string           `json:"slug"`
	Name                  string           `json:"name"`
	License               string           `json:"license"`
	IsAirgap              bool             `json:"isAirgap"`
	CurrentSequence       int64            `json:"currentSequence"`
	UpstreamURI           string           `json:"upstreamUri"`
	IconURI               string           `json:"iconUri"`
	UpdatedAt             *time.Time       `json:"createdAt"`
	CreatedAt             time.Time        `json:"updatedAt"`
	LastUpdateCheckAt     string           `json:"lastUpdateCheckAt"`
	HasPreflight          bool             `json:"hasPreflight"`
	IsConfigurable        bool             `json:"isConfigurable"`
	SnapshotTTL           string           `json:"snapshotTtl"`
	SnapshotSchedule      string           `json:"snapshotSchedule"`
	RestoreInProgressName string           `json:"restoreInProgressName"`
	RestoreUndeployStatus UndeployStatus   `json:"restoreUndeloyStatus"`
	UpdateCheckerSpec     string           `json:"updateCheckerSpec"`
	SemverAutoDeploy      SemverAutoDeploy `json:"semverAutoDeploy"`
	IsGitOps              bool             `json:"isGitOps"`
	InstallState          string           `json:"installState"`
	LastLicenseSync       string           `json:"lastLicenseSync"`
}
