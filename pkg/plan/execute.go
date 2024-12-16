package plan

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/operator"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/store"
)

func Execute(p *types.Plan) error {
	if p == nil {
		return nil
	}

	for _, step := range p.Steps {
		switch step.Type {
		case types.StepTypeAppUpgrade:
			if err := executeAppUpgrade(step); err != nil {
				return err
			}
		default:
			return errors.Errorf("unknown step type %q", step.Type)
		}
	}

	return nil
}

func executeAppUpgrade(step types.PlanStep) error {
	in, ok := step.Input.(types.PlanStepInputAppUpgrade)
	if !ok {
		return errors.New("invalid input for app upgrade step")
	}

	a, err := store.GetStore().GetAppFromSlug(in.AppSlug)
	if err != nil {
		return errors.Wrap(err, "failed to get app from slug")
	}

	source := "Upstream Update"
	if a.IsAirgap {
		source = "Airgap Update"
	}

	// TODO (@salah): pulling the archive should be moved to the config/preflight step
	appArchive, baseSequence, err := pullAppArchive(in)
	if err != nil {
		return errors.Wrap(err, "pull app archive")
	}

	deployOpts := operator.DeployEC2AppOptions{
		AppID:                        a.ID,
		AppSlug:                      a.Slug,
		AppVersionArchive:            appArchive,
		BaseSequence:                 baseSequence,
		VersionLabel:                 in.VersionLabel,
		Source:                       source,
		IsAirgap:                     a.IsAirgap,
		ChannelID:                    in.ChannelID,
		UpdateCursor:                 in.UpdateCursor,
		SkipPreflights:               false,
		ContinueWithFailedPreflights: false,
		PreflightResult:              "",
	}

	if err := operator.MustGetOperator().DeployEC2App(deployOpts); err != nil {
		return errors.Wrap(err, "deploy app")
	}

	return nil
}
