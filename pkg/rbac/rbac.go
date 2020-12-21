package rbac

import (
	"github.com/replicatedhq/kots/pkg/rbac/types"
)

const (
	ClusterAdminRoleID = "cluster-admin"
)

var (
	ClusterAdminRole = types.Role{
		ID:          "cluster-admin",
		Name:        "Cluster Admin",
		Description: "Read/write access to all resources",
		Allow:       []types.Policy{PolicyAllowAll},
	}

	SupportRole = types.Role{
		ID:          "support",
		Name:        "Support",
		Description: "Role for support personnel",
		Allow: []types.Policy{
			PolicyReadonly,
			{Action: "**", Resource: "preflight.*"},
			{Action: "**", Resource: "**.preflight.*"},
			{Action: "**", Resource: "supportbundle.*"},
			{Action: "**", Resource: "**.supportbundle.*"},
		},
		Deny: []types.Policy{
			{Action: "**", Resource: "app.*.downstream.filetree."},
		},
	}

	PolicyAllowAll = types.Policy{
		Name:     "Allow All",
		Action:   "**",
		Resource: "**",
	}

	PolicyReadonly = types.Policy{
		Name:     "Read Only",
		Action:   "read",
		Resource: "**",
	}
)

func DefaultRoles() []types.Role {
	return []types.Role{
		ClusterAdminRole,
		SupportRole,
	}
}
