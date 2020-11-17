package rbac

import (
	"github.com/replicatedhq/kots/pkg/rbac/types"
)

const (
	WildcardGroupID    = "*"
	ClusterAdminRoleID = "cluster-admin"
)

var (
	DefaultGroups   = []types.Group{DefaultGroup}
	DefaultRoles    = []types.Role{ClusterAdminRole}
	DefaultPolicies = []types.Policy{PolicyAllowAll}

	DefaultGroup = types.Group{
		ID:      WildcardGroupID,
		RoleIDs: []string{ClusterAdminRoleID},
	}

	ClusterAdminRole = types.Role{
		ID:          ClusterAdminRoleID,
		Name:        "Cluster Admin",
		Description: "Read/write access to all resounces",
		PolicyIDs:   []string{"allow-all"},
	}

	PolicyAllowAll = types.Policy{
		ID: "allow-all",
		Allowed: []string{
			"**/*",
		},
	}
)
