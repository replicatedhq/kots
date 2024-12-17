package types

type Plan struct {
	ID           string
	AppID        string
	AppSlug      string
	VersionLabel string
	UpdateCursor string
	ChannelID    string
	Steps        []*PlanStep
}

type PlanStep struct {
	ID                string
	Name              string
	Type              PlanStepType
	Status            PlanStepStatus
	StatusDescription string
	Owner             PlanStepOwner
	OwnerHost         string
	Output            interface{}
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
	StepStatusRunning  PlanStepStatus = "running"
	StepStatusComplete PlanStepStatus = "complete"
)

type PlanStepOwner string

const (
	StepOwnerKOTS      PlanStepOwner = "kots"
	StepOwnerECManager PlanStepOwner = "manager"
)
