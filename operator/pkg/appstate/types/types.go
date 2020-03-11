package types

import (
	"errors"
	"regexp"
	"time"
)

var (
	StateReady       State = "ready"
	StateDegraded    State = "degraded"
	StateUnavailable State = "unavailable"
	StateMissing     State = "missing"

	StatusInformerRegexp = regexp.MustCompile(`^(?:([^\/]+)\/)?([^\/]+)\/([^\/]+)$`)
)

type StatusInformerString string

type StatusInformer struct {
	Kind      string
	Name      string
	Namespace string
}

func (s StatusInformerString) Parse() (i StatusInformer, err error) {
	matches := StatusInformerRegexp.FindStringSubmatch(string(s))
	if len(matches) != 4 {
		err = errors.New("status informer format string incorrect")
		return
	}
	i.Namespace = matches[1]
	i.Kind = matches[2]
	i.Name = matches[3]
	return
}

type AppStatus struct {
	AppID          string         `json:"app_id"`
	ResourceStates ResourceStates `json:"resource_states" hash:"set"`
	UpdatedAt      time.Time      `json:"updated_at" hash:"ignore"`
}

type ResourceStates []ResourceState

type ResourceState struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	State     State  `json:"state"`
}

type State string

func MinState(ss ...State) (min State) {
	if len(ss) == 0 {
		return StateMissing
	}
	for _, s := range ss {
		if s == StateMissing || min == StateMissing {
			return StateMissing
		} else if s == StateUnavailable || min == StateUnavailable {
			min = StateUnavailable
		} else if s == StateDegraded || min == StateDegraded {
			min = StateDegraded
		} else if s == StateReady || min == StateReady {
			min = StateReady
		}
	}
	return
}

func (a ResourceStates) Len() int {
	return len(a)
}

func (a ResourceStates) Less(i, j int) bool {
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

func (a ResourceStates) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
