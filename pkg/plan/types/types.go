package types

type Plan struct {
	Steps []PlanStep
}

type PlanStep struct {
	Type              PlanStepType
	Input             interface{}
	Status            PlanStepStatus
	StatusDescription string
	Owner             PlanStepOwner
	OwnerHost         string
	Output            interface{}
}

type PlanStepType string

const (
	StepTypeAppUpgrade PlanStepType = "app_upgrade"
)

type PlanStepStatus string

const (
	StepStatusPending PlanStepStatus = "pending"
)

type PlanStepOwner string

const (
	StepOwnerKOTS PlanStepOwner = "kots"
)

type PlanStepInputAppUpgrade struct {
	AppSlug      string
	VersionLabel string
	UpdateCursor string
	ChannelID    string
}
