package embeddedcluster

import (
	"testing"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-operator/api/v1beta1"
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

func Test_getRoleLabelsImpl(t *testing.T) {
	tests := []struct {
		name   string
		config *embeddedclusterv1beta1.ConfigSpec
		roles  []string
		want   []string
	}{
		{
			name:   "no config",
			config: nil,
			roles:  []string{"a", "b", "c"},
			want:   []string{},
		},
		{
			name: "no roles",
			config: &embeddedclusterv1beta1.ConfigSpec{
				Controller: embeddedclusterv1beta1.NodeRole{
					Name: "a",
				},
			},
			roles: []string{"a", "b", "c"},
			want:  []string{},
		},
		{
			name: "roles with labels",
			config: &embeddedclusterv1beta1.ConfigSpec{
				Controller: embeddedclusterv1beta1.NodeRole{
					Name: "a",
					Labels: map[string]string{
						"a-role": "a-role",
					},
				},
				Custom: []embeddedclusterv1beta1.NodeRole{
					{
						Name: "b",
						Labels: map[string]string{
							"b-role":  "b-role",
							"b-role2": "b-role2",
						},
					},
					{
						Name: "c", // no labels for c
					},
					{
						Name: "d", // d is not in the list of roles to make labels for
						Labels: map[string]string{
							"d-role":  "d-role",
							"d-role2": "d-role2",
						},
					},
				},
			},
			roles: []string{"a", "b", "c"},
			want:  []string{"a-role=a-role", "b-role2=b-role2", "b-role=b-role"},
		},
		{
			name: "roles with labels with bad characters",
			config: &embeddedclusterv1beta1.ConfigSpec{
				Controller: embeddedclusterv1beta1.NodeRole{
					Name: "a",
					Labels: map[string]string{
						"a-role": "this is the a role",
					},
				},
				Custom: []embeddedclusterv1beta1.NodeRole{
					{
						Name: "b",
						Labels: map[string]string{
							"b-role":  " this is the b role ",
							"b-role2": "This! Is! The! Second! B! Role!",
						},
					},
					{
						Name: "c", // no labels for c
					},
					{
						Name: "d", // d is not in the list of roles to make labels for
						Labels: map[string]string{
							"d-role":  "d-role",
							"d-role2": "d-role2",
						},
					},
				},
			},
			roles: []string{"a", "b", "c"},
			want:  []string{"a-role=this-is-the-a-role", "b-role2=This-Is-The-Second-B-Role", "b-role=this-is-the-b-role"},
		},
		{
			name: "roles more than 63 character labels",
			config: &embeddedclusterv1beta1.ConfigSpec{
				Controller: embeddedclusterv1beta1.NodeRole{
					Name: "a",
					Labels: map[string]string{
						"this is a more than 63 character label with a lot of filler to ensure that": "this is a more than 63 character value with a lot of filler to ensure that",
					},
				},
			},
			roles: []string{"a"},
			want:  []string{"this-is-a-more-than-63-character-label-with-a-lot-of-filler-to=this-is-a-more-than-63-character-value-with-a-lot-of-filler-to"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got := getRoleLabelsImpl(tt.config, tt.roles)
			req.Equal(tt.want, got)
		})
	}
}
