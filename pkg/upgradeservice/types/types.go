package types

import (
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
)

type UpgradeServiceParams struct {
	Port string `yaml:"port" json:"port"`

	AppID       string `yaml:"appId" json:"appId"`
	AppSlug     string `yaml:"appSlug" json:"appSlug"`
	AppName     string `yaml:"appName" json:"appName"`
	AppIsAirgap bool   `yaml:"appIsAirgap" json:"appIsAirgap"`
	AppIsGitOps bool   `yaml:"appIsGitOps" json:"appIsGitOps"`
	AppLicense  string `yaml:"appLicense" json:"appLicense"`
	AppArchive  string `yaml:"appArchive" json:"appArchive"`

	Source       string `yaml:"source" json:"source"`
	BaseSequence int64  `yaml:"baseSequence" json:"baseSequence"`
	NextSequence int64  `yaml:"nextSequence" json:"nextSequence"`

	UpdateVersionLabel string `yaml:"updateVersionLabel" json:"updateVersionLabel"`
	UpdateCursor       string `yaml:"updateCursor" json:"updateCursor"`
	UpdateChannelID    string `yaml:"updateChannelID" json:"updateChannelID"`
	UpdateECVersion    string `yaml:"updateECVersion" json:"updateECVersion"`
	UpdateKOTSBin      string `yaml:"updateKotsBin" json:"updateKotsBin"`
	UpdateAirgapBundle string `yaml:"updateAirgapBundle" json:"updateAirgapBundle"`

	CurrentECVersion string `yaml:"currentECVersion" json:"currentECVersion"`

	RegistryEndpoint   string `yaml:"registryEndpoint" json:"registryEndpoint"`
	RegistryUsername   string `yaml:"registryUsername" json:"registryUsername"`
	RegistryPassword   string `yaml:"registryPassword" json:"registryPassword"`
	RegistryNamespace  string `yaml:"registryNamespace" json:"registryNamespace"`
	RegistryIsReadOnly bool   `yaml:"registryIsReadOnly" json:"registryIsReadOnly"`

	ReportingInfo *reportingtypes.ReportingInfo `yaml:"reportingInfo" json:"reportingInfo"`
}
