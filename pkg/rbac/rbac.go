package rbac

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/rbac/types"
)

const (
	ClusterAdminRoleID = "cluster-admin"
)

var (
	DefaultAllowRolePolicies = map[string][]types.Policy{
		ClusterAdminRole.ID: ClusterAdminRole.Allow,
		SupportRole.ID:      SupportRole.Allow,
	}
	DefaultDenyRolePolicies = map[string][]types.Policy{
		ClusterAdminRole.ID: ClusterAdminRole.Deny,
		SupportRole.ID:      SupportRole.Deny,
	}

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

func GetAppAdminRole(appSlug string) types.Role {
	return types.Role{
		ID:          fmt.Sprintf("app-%s-admin", appSlug),
		Name:        fmt.Sprintf("App %s admin", appSlug),
		Description: fmt.Sprintf("Read/write access to all resources for app %s", appSlug),
		Allow: []types.Policy{
			{Action: "read", Resource: "app."},
			{Action: "**", Resource: fmt.Sprintf("app.%s", appSlug)},
			{Action: "**", Resource: fmt.Sprintf("app.%s.**", appSlug)},
		},
	}
}

func GetAppReadonlyRole(appSlug string) types.Role {
	return types.Role{
		ID:          fmt.Sprintf("app-%s-readonly", appSlug),
		Name:        fmt.Sprintf("App %s read-only", appSlug),
		Description: fmt.Sprintf("Read-only access to all resources for app %s", appSlug),
		Allow: []types.Policy{
			{Action: "read", Resource: "app."},
			{Action: "read", Resource: fmt.Sprintf("app.%s", appSlug)},
			{Action: "read", Resource: fmt.Sprintf("app.%s.**", appSlug)},
		},
	}
}
