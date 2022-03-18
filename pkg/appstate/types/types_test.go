package types

import (
	"reflect"
	"testing"
)

func TestStatusInformerString_Parse(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		want    StatusInformer
		wantErr bool
	}{
		{
			name: "kind/name",
			str:  "deploy/sentry-web",
			want: StatusInformer{
				Kind: "deploy",
				Name: "sentry-web",
			},
		},
		{
			name: "namespace/kind/name",
			str:  "default/deploy/sentry-web",
			want: StatusInformer{
				Namespace: "default",
				Kind:      "deploy",
				Name:      "sentry-web",
			},
		},
		{
			name:    "no match",
			str:     "sentry-web",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StatusInformerString(tt.str).Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("StatusInformerString.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StatusInformerString.Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinState(t *testing.T) {
	tests := []struct {
		name string
		ss   []State
		want State
	}{
		{
			name: "ready",
			ss:   []State{StateReady, StateReady},
			want: StateReady,
		},
		{
			name: "updating",
			ss:   []State{StateUpdating, StateReady},
			want: StateUpdating,
		},
		{
			name: "degraded",
			ss:   []State{StateReady, StateDegraded, StateUpdating, StateReady},
			want: StateDegraded,
		},
		{
			name: "unavailable",
			ss:   []State{StateUnavailable, StateDegraded, StateUpdating, StateReady},
			want: StateUnavailable,
		},
		{
			name: "missing",
			ss:   []State{StateUnavailable, StateDegraded, StateMissing, StateUpdating, StateReady},
			want: StateMissing,
		},
		{
			name: "none",
			ss:   []State{},
			want: StateMissing,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MinState(tt.ss...); got != tt.want {
				t.Errorf("MinState() = %v, want %v", got, tt.want)
			}
		})
	}
}
