package types

import "time"

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
