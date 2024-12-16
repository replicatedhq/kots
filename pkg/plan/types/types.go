package types

import (
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
)

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
	StepTypeUpgradeService PlanStepType = "upgrade_service"
	StepTypeAppUpgrade     PlanStepType = "app_upgrade"
	StepTypeECUpgrade      PlanStepType = "ec_upgrade"
)

type PlanStepStatus string

const (
	StepStatusPending PlanStepStatus = "pending"
)

type PlanStepOwner string

const (
	StepOwnerKOTS      PlanStepOwner = "kots"
	StepOwnerECManager PlanStepOwner = "manager"
)

type PlanStepInputUpgradeService struct {
	AppSlug      string
	VersionLabel string
	UpdateCursor string
	ChannelID    string
}

type PlanStepInputAppUpgrade struct {
	AppSlug      string
	VersionLabel string
	UpdateCursor string
	ChannelID    string
	AppArchive   string
	BaseSequence int64
}

type PlanStepInputECUpgrade struct {
	AppSlug   string
	Artifacts *ecv1beta1.ArtifactsLocation
	ECConfig  *ecv1beta1.ConfigSpec
}
