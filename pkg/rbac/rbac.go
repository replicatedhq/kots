package rbac

import (
	"github.com/replicatedhq/kots/pkg/rbac/types"
)

const (
	WildcardGroupID    = "*"
	ClusterAdminRoleID = "cluster-admin"
)

var (
	DefaultGroups = []types.Group{DefaultGroup}

	DefaultAllowRolePolicies = map[string][]types.Policy{
		ClusterAdminRole.ID: ClusterAdminRole.Allow,
	}
	DefaultDenyRolePolicies = map[string][]types.Policy{
		ClusterAdminRole.ID: ClusterAdminRole.Deny,
	}

	DefaultGroup = types.Group{
		ID:      WildcardGroupID,
		RoleIDs: []string{ClusterAdminRole.ID},
	}

	ClusterAdminRole = types.Role{
		ID:          "cluster-admin",
		Name:        "Cluster Admin",
		Description: "Read/write access to all resources",
		Allow:       []types.Policy{PolicyAllowAll},
	}

	PolicyAllowAll = types.Policy{
		Name:     "Allow All",
		Action:   "**",
		Resource: "**",
	}
)
