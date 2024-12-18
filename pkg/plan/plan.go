package plan

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/segmentio/ksuid"
)

var planMutex sync.Mutex

type PlanUpgradeOptions struct {
	AppSlug      string
	VersionLabel string
	UpdateCursor string
	ChannelID    string
}

func PlanUpgrade(s store.Store, opts PlanUpgradeOptions) (*types.Plan, error) {
	a, err := s.GetAppFromSlug(opts.AppSlug)
	if err != nil {
		return nil, errors.Wrap(err, "get app from slug")
	}

	ecVersion, err := getECVersionForRelease(a, opts.VersionLabel, opts.ChannelID, opts.UpdateCursor)
	if err != nil {
		return nil, errors.Wrap(err, "get kots version for release")
	}

	plan := types.Plan{
		ID:           ksuid.New().String(),
		AppID:        a.ID,
		AppSlug:      opts.AppSlug,
		VersionLabel: opts.VersionLabel,
		UpdateCursor: opts.UpdateCursor,
		ChannelID:    opts.ChannelID,
		ECVersion:    ecVersion,
		Steps:        []*types.PlanStep{},
	}

	// app upgrade service
	plan.Steps = append(plan.Steps, &types.PlanStep{
		ID:                ksuid.New().String(),
		Name:              "App Upgrade Service",
		Type:              types.StepTypeAppUpgradeService,
		Status:            types.StepStatusPending,
		StatusDescription: "Pending",
		Owner:             types.StepOwnerKOTS,
	})

	// embedded cluster upgrade
	if ecVersion != util.EmbeddedClusterVersion() {
		plan.Steps = append(plan.Steps, &types.PlanStep{
			ID:                ksuid.New().String(),
			Name:              "Embedded Cluster Upgrade",
			Type:              types.StepTypeECUpgrade,
			Status:            types.StepStatusPending,
			StatusDescription: "Pending embedded cluster upgrade",
			Owner:             types.StepOwnerECManager,
		})
	}

	// app upgrade
	plan.Steps = append(plan.Steps, &types.PlanStep{
		ID:                ksuid.New().String(),
		Name:              "Application Upgrade",
		Type:              types.StepTypeAppUpgrade,
		Status:            types.StepStatusPending,
		StatusDescription: "Pending application upgrade",
		Owner:             types.StepOwnerKOTS,
	})

	return &plan, nil
}

func Resume(s store.Store) error {
	apps, err := s.ListInstalledApps()
	if err != nil {
		return errors.Wrap(err, "list installed apps")
	}
	if len(apps) == 0 {
		return nil
	}
	if len(apps) > 1 {
		return errors.New("more than one app is installed")
	}

	p, _, err := s.GetCurrentPlan(apps[0].ID)
	if err != nil {
		return errors.Wrap(err, "get active plan")
	}
	if p == nil || p.HasEnded() {
		return nil
	}

	go func() {
		if err := Execute(s, p); err != nil {
			logger.Error(errors.Wrap(err, "execute plan"))
		}
	}()

	return nil
}

// TODO (@salah): make each step report status
func Execute(s store.Store, p *types.Plan) (finalError error) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	go startPlanMonitor(s, p, stopCh)

	for _, step := range p.Steps {
		if err := executeStep(s, p, step); err != nil {
			return errors.Wrap(err, "execute step")
		}
	}

	return nil
}

func startPlanMonitor(s store.Store, p *types.Plan, stopCh chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		case <-time.After(time.Second * 2):
			updated, err := s.GetPlan(p.AppID, p.VersionLabel)
			if err != nil {
				logger.Error(errors.Wrap(err, "get plan"))
				continue
			}
			*p = *updated
		}
	}
}

func executeStep(s store.Store, p *types.Plan, step *types.PlanStep) (finalError error) {
	defer func() {
		if finalError != nil {
			if err := markStepFailed(s, p, step.ID, finalError); err != nil {
				logger.Error(errors.Wrap(err, "mark step failed"))
			}
		}
	}()

	switch step.Status {
	case types.StepStatusFailed:
		return errors.Errorf("step has already failed. status: %q. description: %q", step.Status, step.StatusDescription)
	case types.StepStatusComplete:
		logger.Infof("Skipping step %q of plan %q because it already completed", step.Name, p.ID)
		return nil
	}

	logger.Infof("Executing step %q of plan %q", step.Name, p.ID)

	switch step.Type {
	case types.StepTypeAppUpgradeService:
		if step.Status != types.StepStatusPending {
			return errors.Errorf("step %q cannot be resumed", step.Name, p.ID)
		}
		if err := executeAppUpgradeService(s, p, step); err != nil {
			return errors.Wrap(err, "execute app upgrade service")
		}
		if err := waitForStep(s, p, step.ID); err != nil {
			return errors.Wrap(err, "wait for upgrade service")
		}

	case types.StepTypeECUpgrade:
		if step.Status == types.StepStatusPending {
			if err := executeECUpgrade(s, p, step); err != nil {
				return errors.Wrap(err, "execute embedded cluster upgrade")
			}
		}
		if err := waitForStep(s, p, step.ID); err != nil {
			return errors.Wrap(err, "wait for embedded cluster upgrade")
		}

	case types.StepTypeAppUpgrade:
		if err := executeAppUpgrade(s, p, step); err != nil {
			return errors.Wrap(err, "execute app upgrade")
		}
	default:
		return errors.Errorf("unknown step type %q", step.Type)
	}

	logger.Infof("Step %q of plan %q completed", step.Name, p.ID)
	return nil
}

func waitForStep(s store.Store, p *types.Plan, stepID string) error {
	for {
		stepIndex := -1
		for i, step := range p.Steps {
			if step.ID == stepID {
				stepIndex = i
				break
			}
		}
		if stepIndex == -1 {
			return errors.Errorf("step %s not found in plan", stepID)
		}

		if p.Steps[stepIndex].Status == types.StepStatusComplete {
			return nil
		}
		if p.Steps[stepIndex].Status == types.StepStatusFailed {
			return errors.Errorf("step failed: %s", p.Steps[stepIndex].StatusDescription)
		}

		time.Sleep(time.Second * 2)
	}
}

func markStepFailed(s store.Store, p *types.Plan, stepID string, err error) error {
	return UpdateStep(s, UpdateStepOptions{
		AppSlug:           p.AppSlug,
		VersionLabel:      p.VersionLabel,
		StepID:            stepID,
		Status:            types.StepStatusFailed,
		StatusDescription: err.Error(),
	})
}

type UpdateStepOptions struct {
	AppSlug           string
	VersionLabel      string
	StepID            string
	Status            types.PlanStepStatus
	StatusDescription string
	Output            string
}

func UpdateStep(s store.Store, opts UpdateStepOptions) error {
	planMutex.Lock()
	defer planMutex.Unlock()

	a, err := s.GetAppFromSlug(opts.AppSlug)
	if err != nil {
		return errors.Wrap(err, "get app from slug")
	}

	p, err := s.GetPlan(a.ID, opts.VersionLabel)
	if err != nil {
		return errors.Wrap(err, "get plan")
	}

	stepIndex := -1
	for i, s := range p.Steps {
		if s.ID == opts.StepID {
			stepIndex = i
			break
		}
	}
	if stepIndex == -1 {
		return errors.Errorf("step %s not found in plan", opts.StepID)
	}

	p.Steps[stepIndex].Status = opts.Status
	p.Steps[stepIndex].StatusDescription = opts.StatusDescription
	p.Steps[stepIndex].Output = opts.Output

	if err := s.UpsertPlan(p); err != nil {
		return errors.Wrap(err, "update plan")
	}

	return nil
}
