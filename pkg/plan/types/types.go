package types

type Plan struct {
	ID           string      `json:"id" yaml:"id"`
	AppID        string      `json:"appId" yaml:"appId"`
	AppSlug      string      `json:"appSlug" yaml:"appSlug"`
	VersionLabel string      `json:"versionLabel" yaml:"versionLabel"`
	UpdateCursor string      `json:"updateCursor" yaml:"updateCursor"`
	ChannelID    string      `json:"channelId" yaml:"channelId"`
	ECVersion    string      `json:"ecVersion" yaml:"ecVersion"`
	Steps        []*PlanStep `json:"steps" yaml:"steps"`
}

type PlanStep struct {
	ID                string         `json:"id" yaml:"id"`
	Name              string         `json:"name" yaml:"name"`
	Type              PlanStepType   `json:"type" yaml:"type"`
	Status            PlanStepStatus `json:"status" yaml:"status"`
	StatusDescription string         `json:"statusDescription" yaml:"statusDescription"`
	Owner             PlanStepOwner  `json:"owner" yaml:"owner"`
	OwnerHost         string         `json:"ownerHost" yaml:"ownerHost"`
	Output            interface{}    `json:"output" yaml:"output"`
}

type PlanStepType string

const (
	StepTypeAppUpgradeService PlanStepType = "app-upgrade-service"
	StepTypeAppUpgrade        PlanStepType = "app-upgrade"
	StepTypeECUpgrade         PlanStepType = "ec-upgrade"
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
