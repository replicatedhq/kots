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
	StepTypeAppUpgradeService PlanStepType = "app_upgrade_service"
	StepTypeAppUpgrade        PlanStepType = "app_upgrade"
	StepTypeECUpgrade         PlanStepType = "ec_upgrade"
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
	status, _ := p.GetStatus()
	return status == StepStatusFailed || status == StepStatusComplete
}

func (p *Plan) GetStatus() (PlanStepStatus, string) {
	return minStatus(p.Steps)
}

func minStatus(steps []*PlanStep) (PlanStepStatus, string) {
	var min PlanStepStatus
	var description string

	for _, s := range steps {
		if s.Status == StepStatusFailed || min == StepStatusFailed {
			return StepStatusFailed, s.StatusDescription
		} else if s.Status == StepStatusStarting || min == StepStatusStarting {
			min = StepStatusStarting
			description = s.StatusDescription
		} else if s.Status == StepStatusRunning || min == StepStatusRunning {
			min = StepStatusRunning
			description = s.StatusDescription
		} else if s.Status == StepStatusPending || min == StepStatusPending {
			min = StepStatusPending
			description = s.StatusDescription
		} else if s.Status == StepStatusComplete || min == StepStatusComplete {
			min = StepStatusComplete
			description = s.StatusDescription
		}
	}

	return min, description
}
