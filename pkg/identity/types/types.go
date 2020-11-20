package types

import (
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
	ingresstypes "github.com/replicatedhq/kots/pkg/ingress/types"
	rbactypes "github.com/replicatedhq/kots/pkg/rbac/types"
)

type Config struct {
	Enabled             bool     `json:"enabled" yaml:"enabled"`
	DisablePasswordAuth bool     `json:"disablePasswordAuth" yaml:"disablePasswordAuth"`
	RestrictedGroups    []string `json:"restrictedGroups,omitempty" yaml:"restrictedGroups,omitempty"`
	EnableAdvancedRBAC  bool     `json:"enableAdvancedRbac,omitempty" yaml:"enableAdvancedRbac,omitempty"`
	RBAC                struct {
		Groups   []rbactypes.Group  `json:"groups,omitempty" yaml:"groups,omitempty"`
		Roles    []rbactypes.Role   `json:"roles,omitempty" yaml:"roles,omitempty"`
		Policies []rbactypes.Policy `json:"policies,omitempty" yaml:"policies,omitempty"`
	} `json:"rbac,omitempty" yaml:"rbac,omitempty"`
	IngressConfig ingresstypes.Config  `json:"ingressConfig" yaml:"ingressConfig"`
	DexConnectors []dextypes.Connector `json:"dexConnectors,omitempty" yaml:"dexConnectors,omitempty"`
}
