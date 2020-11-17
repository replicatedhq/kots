package handlers

import (
	"testing"

	"github.com/replicatedhq/kots/kotsadm/pkg/session/types"
)

func Test_sessionAuthorize(t *testing.T) {
	tests := []struct {
		name  string
		roles []types.SessionRole
		want  bool
	}{
		{
			name: "allowed",
			roles: []types.SessionRole{{
				Policies: []types.SessionPolicy{{
					Allowed: []string{"**/*"},
				}},
			}},
			want: true,
		},
		{
			name: "denied",
			roles: []types.SessionRole{{
				Policies: []types.SessionPolicy{{
					Denied: []string{"**/*"},
				}},
			}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionAuthorize(tt.roles); got != tt.want {
				t.Errorf("sessionAuthorize() = %v, want %v", got, tt.want)
			}
		})
	}
}
