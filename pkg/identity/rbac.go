package identity

import (
	"github.com/replicatedhq/kots/pkg/rbac"
	rbactypes "github.com/replicatedhq/kots/pkg/rbac/types"
)

func RestrictedGroupsToRBACGroups(restrictedGroups []string) []rbactypes.Group {
	groups := []rbactypes.Group{}
	for _, restrictedGroup := range restrictedGroups {
		groups = append(groups, rbactypes.Group{
			ID:      restrictedGroup,
			RoleIDs: []string{rbac.ClusterAdminRoleID},
		})
	}
	return groups
}
