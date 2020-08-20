package appstatus

import (
	"github.com/replicatedhq/kots/kotsadm/pkg/appstatus/types"
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
