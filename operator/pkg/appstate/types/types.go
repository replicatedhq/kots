package types

var (
	StateReady       State = "ready"
	StateDegraded    State = "degraded"
	StateUnavailable State = "unavailable"
	StateMissing     State = "missing"
)

type StatusInformer struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type AppStatus []ResourceState

type ResourceState struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	State     State  `json:"state"`
}

type State string

func (a AppStatus) Len() int {
	return len(a)
}

func (a AppStatus) Less(i, j int) bool {
	if a[i].Kind < a[j].Kind {
		return true
	}
	if a[i].Name < a[j].Name {
		return true
	}
	if a[i].Namespace < a[j].Namespace {
		return true
	}
	return false
}

func (a AppStatus) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
