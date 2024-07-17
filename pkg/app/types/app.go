package types

import (
	"time"

	"github.com/replicatedhq/kots/pkg/util"
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
	UpdatedAt             *time.Time     `json:"updatedAt"`
	CreatedAt             time.Time      `json:"createdAt"`
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
	ChannelID             string         `json:"channel_id"`
}

func (a *App) GetID() string {
	return a.ID
}

func (a *App) GetSlug() string {
	return a.Slug
}

func (a *App) GetCurrentSequence() int64 {
	return a.CurrentSequence
}

func (a *App) GetChannelID() string {
	return a.ChannelID
}

func (a *App) GetIsAirgap() bool {
	return a.IsAirgap
}

func (a *App) GetNamespace() string {
	return util.PodNamespace
}
