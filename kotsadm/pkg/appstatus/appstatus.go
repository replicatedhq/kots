package appstatus

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

type AppStatus struct {
	AppID          string          `json:"appId"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	ResourceStates []ResourceState `json:"resourceStates"`
	State          State           `json:"state"`
}

type ResourceState struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	State     State  `json:"state"`
}

type State string

const (
	StateReady       State = "ready"
	StateDegraded    State = "degraded"
	StateUnavailable State = "unavailable"
	StateMissing     State = "missing"
)

func Get(appID string) (*AppStatus, error) {
	db := persistence.MustGetPGSession()
	query := `select resource_states, updated_at from app_status where app_id = $1`
	row := db.QueryRow(query, appID)

	var updatedAt sql.NullTime
	var resourceStatesStr sql.NullString

	if err := row.Scan(&resourceStatesStr, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return &AppStatus{
				AppID:          appID,
				UpdatedAt:      time.Time{},
				ResourceStates: []ResourceState{},
				State:          StateMissing,
			}, nil
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	appStatus := AppStatus{
		AppID: appID,
	}

	if updatedAt.Valid {
		appStatus.UpdatedAt = updatedAt.Time
	}

	if resourceStatesStr.Valid {
		var resourceStates []ResourceState
		if err := json.Unmarshal([]byte(resourceStatesStr.String), &resourceStates); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal resource states")
		}
		appStatus.ResourceStates = resourceStates
	}

	appStatus.State = getState(appStatus.ResourceStates)

	return &appStatus, nil
}

func getState(resourceStates []ResourceState) State {
	if len(resourceStates) == 0 {
		return StateMissing
	}
	max := StateReady
	for _, resourceState := range resourceStates {
		max = minState(max, resourceState.State)
	}
	return max
}

func minState(a State, b State) State {
	if a == StateMissing || b == StateMissing {
		return StateMissing
	}
	if a == StateUnavailable || b == StateUnavailable {
		return StateUnavailable
	}
	if a == StateDegraded || b == StateDegraded {
		return StateDegraded
	}
	if a == StateReady || b == StateReady {
		return StateReady
	}
	return StateMissing
}
