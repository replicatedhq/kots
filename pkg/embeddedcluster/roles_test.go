package embeddedcluster

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortRoles(t *testing.T) {
	tests := []struct {
		name           string
		controllerRole string
		inputRoles     []string
		want           []string
	}{
		{
			name:           "no controller",
			controllerRole: "",
			inputRoles:     []string{"c", "b", "a"},
			want:           []string{"a", "b", "c"},
		},
		{
			name:           "controller in front",
			controllerRole: "controller",
			inputRoles:     []string{"controller", "c", "b"},
			want:           []string{"controller", "b", "c"},
		},
		{
			name:           "controller in middle",
			controllerRole: "controller",
			inputRoles:     []string{"c", "controller", "b"},
			want:           []string{"controller", "b", "c"},
		},
		{
			name:           "controller at end",
			controllerRole: "controller",
			inputRoles:     []string{"c", "b", "controller"},
			want:           []string{"controller", "b", "c"},
		},
		{
			name:           "controller not present",
			controllerRole: "controller",
			inputRoles:     []string{"c", "b", "d"},
			want:           []string{"b", "c", "d"},
		},
		{
			name:           "other controller name",
			controllerRole: "other-controller",
			inputRoles:     []string{"c", "b", "a"},
			want:           []string{"a", "b", "c"},
		},
		{
			name:           "other controller name in list",
			controllerRole: "other-controller",
			inputRoles:     []string{"c", "b", "other-controller"},
			want:           []string{"other-controller", "b", "c"},
		},
		{
			name:           "more items",
			controllerRole: "controller",
			inputRoles:     []string{"a", "b", "c", "controller", "e", "f", "x", "y", "z", "g", "h", "i", "j", "k", "l", "m", "n"},
			want:           []string{"controller", "a", "b", "c", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "x", "y", "z"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got := SortRoles(tt.controllerRole, tt.inputRoles)
			req.Equal(tt.want, got)
		})
	}
}
