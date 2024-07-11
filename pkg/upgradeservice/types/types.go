package types

import (
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
)

type UpgradeServiceParams struct {
	Port string `yaml:"port"`

	AppID       string `yaml:"appId"`
	AppSlug     string `yaml:"appSlug"`
	AppName     string `yaml:"appName"`
	AppIsAirgap bool   `yaml:"appIsAirgap"`
	AppIsGitOps bool   `yaml:"appIsGitOps"`
	AppLicense  string `yaml:"appLicense"`
	AppArchive  string `yaml:"appArchive"`

	Source       string `yaml:"source"`
	BaseSequence int64  `yaml:"baseSequence"`
	NextSequence int64  `yaml:"nextSequence"`

	UpdateVersionLabel string `yaml:"updateVersionLabel"`
	UpdateCursor       string `yaml:"updateCursor"`
	UpdateChannelID    string `yaml:"updateChannelID"`
	UpdateECVersion    string `yaml:"updateECVersion"`
	UpdateKOTSBin      string `yaml:"updateKotsBin"`
	UpdateAirgapBundle string `yaml:"updateAirgapBundle"`

	CurrentECVersion string `yaml:"currentECVersion"`

	RegistryEndpoint   string `yaml:"registryEndpoint"`
	RegistryUsername   string `yaml:"registryUsername"`
	RegistryPassword   string `yaml:"registryPassword"`
	RegistryNamespace  string `yaml:"registryNamespace"`
	RegistryIsReadOnly bool   `yaml:"registryIsReadOnly"`

	ReportingInfo *reportingtypes.ReportingInfo `yaml:"reportingInfo"`
}
