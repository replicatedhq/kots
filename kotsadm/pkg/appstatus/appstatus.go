package appstatus

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/api/appstatus/types"
)

func GetState(resourceStates []types.ResourceState) types.State {
	if len(resourceStates) == 0 {
		return types.StateMissing
	}
	max := types.StateReady
	for _, resourceState := range resourceStates {
		max = minState(max, resourceState.State)
	}
	return max
}

func minState(a types.State, b types.State) types.State {
	if a == types.StateMissing || b == types.StateMissing {
		return types.StateMissing
	}
	if a == types.StateUnavailable || b == types.StateUnavailable {
		return types.StateUnavailable
	}
	if a == types.StateDegraded || b == types.StateDegraded {
		return types.StateDegraded
	}
	if a == types.StateReady || b == types.StateReady {
		return types.StateReady
	}
	return types.StateMissing
}

func Set(appID string, resourceStates []types.ResourceState, updatedAt time.Time) error {
	marshalledResourceStates, err := json.Marshal(resourceStates)
	if err != nil {
		return errors.Wrap(err, "failed to json marshal resource states")
	}

	db := persistence.MustGetPGSession()
	query := `insert into app_status (app_id, resource_states, updated_at) values ($1, $2, $3) on conflict (app_id) do update set resource_states = $2, updated_at = $3`
	_, err = db.Exec(query, appID, marshalledResourceStates, updatedAt)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
