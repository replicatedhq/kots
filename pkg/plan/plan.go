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
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var planMutex sync.Mutex

type PlanUpgradeOptions struct {
	AppSlug      string
	VersionLabel string
	UpdateCursor string
	ChannelID    string
}

func PlanUpgrade(s store.Store, kcli kbclient.Client, opts PlanUpgradeOptions) (*types.Plan, error) {
	a, err := s.GetAppFromSlug(opts.AppSlug)
	if err != nil {
		return nil, errors.Wrap(err, "get app from slug")
	}

	manifests, err := getReleaseManifests(a, opts.VersionLabel, opts.ChannelID, opts.UpdateCursor)
	if err != nil {
		return nil, errors.Wrap(err, "get release manifests")
	}

	newECConfigSpec, err := findECConfigSpecInRelease(manifests)
	if err != nil {
		return nil, errors.Wrap(err, "find embedded cluster config in release")
	}

	p := types.Plan{
		ID:               ksuid.New().String(),
		AppID:            a.ID,
		AppSlug:          opts.AppSlug,
		VersionLabel:     opts.VersionLabel,
		UpdateCursor:     opts.UpdateCursor,
		ChannelID:        opts.ChannelID,
		CurrentECVersion: util.EmbeddedClusterVersion(),
		NewECVersion:     newECConfigSpec.Version,
		IsAirgap:         a.IsAirgap,
		Steps:            []*types.PlanStep{},
	}

	// app upgrade service
	ausSteps, err := planAppUpgradeService(s, &p)
	if err != nil {
		return nil, errors.Wrap(err, "plan app upgrade service")
	}
	p.Steps = append(p.Steps, ausSteps...)

	// ec managers upgrade
	ecManagerSteps, err := planECManagersUpgrade(kcli, a, newECConfigSpec.Version)
	if err != nil {
		return nil, errors.Wrap(err, "plan ec managers upgrade")
	}
	p.Steps = append(p.Steps, ecManagerSteps...)

	// TODO (@salah) implement our EC addons upgrade. make sure kots gets upgraded first.

	// k0s upgrade
	k0sUpgradeSteps, err := planK0sUpgrade(s, kcli, a, opts.VersionLabel, newECConfigSpec)
	if err != nil {
		return nil, errors.Wrap(err, "plan k0s upgrade")
	}
	p.Steps = append(p.Steps, k0sUpgradeSteps...)

	// ec extensions
	ecExtensionSteps, err := planECExtensions(kcli, newECConfigSpec)
	if err != nil {
		return nil, errors.Wrap(err, "plan ec extensions")
	}
	p.Steps = append(p.Steps, ecExtensionSteps...)

	// app upgrade
	appUpgradeSteps, err := planAppUpgrade()
	if err != nil {
		return nil, errors.Wrap(err, "plan app upgrade")
	}
	p.Steps = append(p.Steps, appUpgradeSteps...)

	return &p, nil
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
			logger.Error(errors.Wrapf(err, "failed to execute plan %s", p.ID))
		}
	}()

	return nil
}

// TODO (@salah): make each step report better status
func Execute(s store.Store, p *types.Plan) error {
	for _, step := range p.Steps {
		if err := executeStep(s, p, step); err != nil {
			return errors.Wrap(err, "execute step")
		}

		updated, err := s.GetPlan(p.AppID, p.VersionLabel)
		if err != nil {
			logger.Errorf("Failed to get plan: %v", err)
			continue
		}
		*p = *updated
	}

	logger.Infof("Plan %q completed successfully", p.ID)

	return nil
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
		if err := executeAppUpgradeService(s, p, step); err != nil {
			return errors.Wrap(err, "execute app upgrade service")
		}

	case types.StepTypeECManagerUpgrade:
		if err := executeECManagerUpgrade(s, p, step); err != nil {
			return errors.Wrap(err, "execute ec manager upgrade")
		}

	case types.StepTypeK0sUpgrade:
		if err := executeK0sUpgrade(s, p, step); err != nil {
			return errors.Wrap(err, "execute k0s upgrade")
		}

	case types.StepTypeECExtensionAdd:
		if err := executeECExtensionAdd(s, p, step); err != nil {
			return errors.Wrap(err, "execute embedded cluster extension add")
		}

	case types.StepTypeECExtensionUpgrade:
		if err := executeECExtensionUpgrade(s, p, step); err != nil {
			return errors.Wrap(err, "execute embedded cluster extension upgrade")
		}

	case types.StepTypeECExtensionRemove:
		if err := executeECExtensionRemove(s, p, step); err != nil {
			return errors.Wrap(err, "execute embedded cluster extension remove")
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
		updated, err := s.GetPlan(p.AppID, p.VersionLabel)
		if err != nil {
			logger.Errorf("Failed to get plan: %v", err)
			time.Sleep(time.Second * 2)
			continue
		}

		stepIndex := -1
		for i, step := range updated.Steps {
			if step.ID == stepID {
				stepIndex = i
				break
			}
		}
		if stepIndex == -1 {
			return errors.Errorf("step %s not found in plan %s", stepID, updated.ID)
		}

		if updated.Steps[stepIndex].Status == types.StepStatusComplete {
			return nil
		}
		if updated.Steps[stepIndex].Status == types.StepStatusFailed {
			return errors.Errorf("step failed: %s", updated.Steps[stepIndex].StatusDescription)
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
