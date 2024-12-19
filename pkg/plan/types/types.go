package types

import (
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	upgradeservicetypes "github.com/replicatedhq/kots/pkg/upgradeservice/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

type Plan struct {
	ID               string      `json:"id" yaml:"id"`
	AppID            string      `json:"appId" yaml:"appId"`
	AppSlug          string      `json:"appSlug" yaml:"appSlug"`
	VersionLabel     string      `json:"versionLabel" yaml:"versionLabel"`
	UpdateCursor     string      `json:"updateCursor" yaml:"updateCursor"`
	ChannelID        string      `json:"channelId" yaml:"channelId"`
	CurrentECVersion string      `json:"currentECVersion" yaml:"currentECVersion"`
	NewECVersion     string      `json:"newECVersion" yaml:"newECVersion"`
	IsAirgap         bool        `json:"isAirgap" yaml:"isAirgap"`
	BaseSequence     int64       `json:"baseSequence" yaml:"baseSequence"`
	NextSequence     int64       `json:"nextSequence" yaml:"nextSequence"`
	Source           string      `json:"source" yaml:"source"`
	Steps            []*PlanStep `json:"steps" yaml:"steps"`
}

type PlanStep struct {
	ID                string         `json:"id" yaml:"id"`
	Name              string         `json:"name" yaml:"name"`
	Type              PlanStepType   `json:"type" yaml:"type"`
	Status            PlanStepStatus `json:"status" yaml:"status"`
	StatusDescription string         `json:"statusDescription" yaml:"statusDescription"`
	Owner             PlanStepOwner  `json:"owner" yaml:"owner"`
	OwnerHost         string         `json:"ownerHost" yaml:"ownerHost"`
	Input             interface{}    `json:"input" yaml:"input"`
	Output            interface{}    `json:"output" yaml:"output"`
}

type PlanStepType string

const (
	StepTypeAppUpgradeService  PlanStepType = "app-upgrade-service"
	StepTypeAppUpgrade         PlanStepType = "app-upgrade"
	StepTypeECUpgrade          PlanStepType = "ec-upgrade"
	StepTypeECExtensionAdd     PlanStepType = "ec-extension-add"
	StepTypeECExtensionUpgrade PlanStepType = "ec-extension-upgrade"
	StepTypeECExtensionRemove  PlanStepType = "ec-extension-remove"
)

type PlanStepStatus string

const (
	StepStatusPending  PlanStepStatus = "pending"
	StepStatusStarting PlanStepStatus = "starting"
	StepStatusRunning  PlanStepStatus = "running"
	StepStatusComplete PlanStepStatus = "complete"
	StepStatusFailed   PlanStepStatus = "failed"
)

type PlanStepOwner string

const (
	StepOwnerKOTS      PlanStepOwner = "kots"
	StepOwnerECManager PlanStepOwner = "manager"
)

type PlanStepInputAppUpgradeService struct {
	Params upgradeservicetypes.UpgradeServiceParams `json:"params" yaml:"params"`
}

type PlanStepInputECUpgrade struct {
	CurrentECInstallation       ecv1beta1.Installation   `json:"currentECInstallation" yaml:"currentECInstallation"`
	CurrentKOTSInstallation     kotsv1beta1.Installation `json:"currentKOTSInstallation" yaml:"currentKOTSInstallation"`
	NewECConfigSpec             ecv1beta1.ConfigSpec     `json:"newECConfigSpec" yaml:"newECConfigSpec"`
	IsDisasterRecoverySupported bool                     `json:"isDisasterRecoverySupported" yaml:"isDisasterRecoverySupported"`
}

type PlanStepInputECExtension struct {
	Repos []k0sv1beta1.Repository `json:"repos" yaml:"repos"`
	Chart ecv1beta1.Chart         `json:"chart" yaml:"chart"`
}

func (p *Plan) HasEnded() bool {
	status := p.GetStatus()
	return status == StepStatusFailed || status == StepStatusComplete
}

func (p *Plan) GetStatus() PlanStepStatus {
	return p.CurrentStep().Status
}

func (p *Plan) CurrentStep() *PlanStep {
	for _, s := range p.Steps {
		if s.Status == StepStatusFailed {
			return s
		}
	}
	for _, s := range p.Steps {
		if s.Status == StepStatusStarting {
			return s
		}
	}
	for _, s := range p.Steps {
		if s.Status == StepStatusRunning {
			return s
		}
	}
	for _, s := range p.Steps {
		if s.Status == StepStatusPending {
			return s
		}
	}
	for _, s := range p.Steps {
		if s.Status == StepStatusComplete {
			return s
		}
	}
	return &PlanStep{}
}
