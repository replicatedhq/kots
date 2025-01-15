package plan

import (
	"encoding/json"

	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/store"
)

type IPlanStep interface {
	Execute(s store.Store, p *types.Plan) error
}

func UnmarshalPlanSteps(marshalled []byte) ([]IPlanStep, error) {
	var ps []*types.PlanStep
	err := json.Unmarshal(marshalled, &ps)
	if err != nil {
		return nil, err
	}
	var result []IPlanStep
	for _, p := range ps {
		switch p.Type {
		case types.StepTypeECUpgrade:
			var pss *PlanStepECUpgrade
			err := json.Unmarshal(marshalled, &pss)
			if err != nil {
				return nil, err
			}
			result = append(result, pss)

		default:
			// TODO
		}
	}

	return result, nil
}
