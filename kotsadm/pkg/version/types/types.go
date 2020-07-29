package types

import (
	v1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"time"
)

type AppVersion struct {
	Sequence     int64                           `json:"sequence"`
	UpdateCursor int                             `json:"updateCursor"`
	VersionLabel string                          `json:"title"`
	Status       string                          `json:"status"`
	CreatedOn    *time.Time                      `json:"createdOn"`
	ReleaseNotes string                          `json:"releaseNotes"`
	DeployedAt   string                          `json:"deployedAt"`
	BackupSpec   string                          `json:"backupSpec"`
	YamlErrors   []v1beta1.InstallationYAMLError `json:"yamlErrors"`
}
