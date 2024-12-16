package plan

import (
	"context"

	"github.com/replicatedhq/kots/pkg/plan/types"
)

type PlanUpgradeOptions struct {
	AppSlug      string
	VersionLabel string
	UpdateCursor string
	ChannelID    string
}

func PlanUpgrade(ctx context.Context, opts PlanUpgradeOptions) (*types.Plan, error) {
	plan := types.Plan{
		Steps: []types.PlanStep{
			{
				Type:              types.StepTypeUpgradeService,
				Status:            types.StepStatusPending,
				StatusDescription: "Pending",
				Owner:             types.StepOwnerKOTS,
				Input: types.PlanStepInputUpgradeService{
					AppSlug:      opts.AppSlug,
					VersionLabel: opts.VersionLabel,
					UpdateCursor: opts.UpdateCursor,
					ChannelID:    opts.ChannelID,
				},
			},
		},
	}

	// TODO (@salah): this should be done by new kots
	// appArchive, _, err := pullAppArchive(opts.AppSlug, opts.VersionLabel, opts.UpdateCursor, opts.ChannelID)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "pull app archive")
	// }

	// TODO (@salah): config plan step should update / re-render the app archive with new config values

	// kotsKinds, err := kotsutil.LoadKotsKinds(appArchive)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "load kots kinds")
	// }

	// if kotsKinds.EmbeddedClusterConfig != nil {
	// 	if kotsKinds.EmbeddedClusterConfig.Spec.Version != util.EmbeddedClusterVersion() {
	// 		plan.Steps = append(plan.Steps, types.PlanStep{
	// 			Type:              types.StepTypeECUpgrade,
	// 			Status:            types.StepStatusPending,
	// 			StatusDescription: "Pending embedded cluster upgrade",
	// 			Owner:             types.StepOwnerECManager,
	// 			Input: types.PlanStepInputECUpgrade{
	// 				AppSlug:  opts.AppSlug,
	// 				ECConfig: &kotsKinds.EmbeddedClusterConfig.Spec,
	// 			},
	// 		})
	// 	}
	// }

	// plan.Steps = append(plan.Steps, types.PlanStep{
	// 	Type:              types.StepTypeAppUpgrade,
	// 	Status:            types.StepStatusPending,
	// 	StatusDescription: "Pending application upgrade",
	// 	Owner:             types.StepOwnerKOTS,
	// 	Input: types.PlanStepInputAppUpgrade{
	// 		AppSlug:      opts.AppSlug,
	// 		VersionLabel: opts.VersionLabel,
	// 		UpdateCursor: opts.UpdateCursor,
	// 		ChannelID:    opts.ChannelID,
	// 		AppArchive:   appArchive,
	// 		BaseSequence: baseSequence,
	// 	},
	// })

	return &plan, nil
}
