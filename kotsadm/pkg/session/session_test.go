package session

import (
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/kotsadm/pkg/session/types"
	"github.com/replicatedhq/kots/pkg/rbac"
	rbactypes "github.com/replicatedhq/kots/pkg/rbac/types"
)

func Test_GetSessionRolesFromRBAC(t *testing.T) {
	type args struct {
		sessionGroupIDs []string
		groups          []rbactypes.Group
		roles           []rbactypes.Role
		policies        []rbactypes.Policy
	}
	tests := []struct {
		name string
		args args
		want []types.SessionRole
	}{
		{
			name: "wildcard",
			args: args{
				sessionGroupIDs: nil,
				groups:          rbac.DefaultGroups,
				roles:           rbac.DefaultRoles,
				policies:        rbac.DefaultPolicies,
			},
			want: []types.SessionRole{
				{
					ID: rbac.ClusterAdminRoleID,
					Policies: []types.SessionPolicy{
						{
							ID:      rbac.PolicyAllowAll.ID,
							Allowed: rbac.PolicyAllowAll.Allowed,
							Denied:  rbac.PolicyAllowAll.Denied,
						},
					},
				},
			},
		},
		{
			name: "custom wildcard",
			args: args{
				sessionGroupIDs: []string{"custom_group"},
				groups: append(rbac.DefaultGroups, rbactypes.Group{
					ID:      "custom_group",
					RoleIDs: []string{"custom_role"},
				}),
				roles: append(rbac.DefaultRoles, rbactypes.Role{
					ID:        "custom_role",
					PolicyIDs: []string{"custom_policy"},
				}),
				policies: append(rbac.DefaultPolicies, rbactypes.Policy{
					ID:      "custom_policy",
					Allowed: []string{"allow"},
					Denied:  []string{"deny"},
				}),
			},
			want: []types.SessionRole{
				{
					ID: rbac.ClusterAdminRoleID,
					Policies: []types.SessionPolicy{
						{
							ID:      rbac.PolicyAllowAll.ID,
							Allowed: rbac.PolicyAllowAll.Allowed,
							Denied:  rbac.PolicyAllowAll.Denied,
						},
					},
				},
				{
					ID: "custom_role",
					Policies: []types.SessionPolicy{
						{
							ID:      "custom_policy",
							Allowed: []string{"allow"},
							Denied:  []string{"deny"},
						},
					},
				},
			},
		},
		{
			name: "custom",
			args: args{
				sessionGroupIDs: []string{"custom_group"},
				groups: []rbactypes.Group{{
					ID:      "custom_group",
					RoleIDs: []string{"custom_role"},
				}},
				roles: append(rbac.DefaultRoles, rbactypes.Role{
					ID:        "custom_role",
					PolicyIDs: []string{"custom_policy"},
				}),
				policies: append(rbac.DefaultPolicies, rbactypes.Policy{
					ID:      "custom_policy",
					Allowed: []string{"allow"},
					Denied:  []string{"deny"},
				}),
			},
			want: []types.SessionRole{
				{
					ID: "custom_role",
					Policies: []types.SessionPolicy{
						{
							ID:      "custom_policy",
							Allowed: []string{"allow"},
							Denied:  []string{"deny"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSessionRolesFromRBAC(tt.args.sessionGroupIDs, tt.args.groups, tt.args.roles, tt.args.policies); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSessionRolesFromRBAC() = %v, want %v", got, tt.want)
			}
		})
	}
}
