package plan

import "github.com/replicatedhq/kots/pkg/plan/types"

type PlanUpgradeOptions struct {
	AppSlug      string
	VersionLabel string
	UpdateCursor string
	ChannelID    string
}

func PlanUpgrade(opts PlanUpgradeOptions) (*types.Plan, error) {
	plan := types.Plan{
		Steps: []types.PlanStep{
			{
				Type:              types.StepTypeAppUpgrade,
				Status:            types.StepStatusPending,
				StatusDescription: "Pending application upgrade",
				Owner:             types.StepOwnerKOTS,
				Input: types.PlanStepInputAppUpgrade{
					AppSlug:      opts.AppSlug,
					VersionLabel: opts.VersionLabel,
					UpdateCursor: opts.UpdateCursor,
					ChannelID:    opts.ChannelID,
				},
			},
		},
	}

	return &plan, nil
}
