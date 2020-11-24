package types

import (
	"github.com/pkg/errors"
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
	ingresstypes "github.com/replicatedhq/kots/pkg/ingress/types"
	rbactypes "github.com/replicatedhq/kots/pkg/rbac/types"
)

type Config struct {
	Enabled                bool                 `json:"enabled" yaml:"enabled"`
	DisablePasswordAuth    bool                 `json:"disablePasswordAuth" yaml:"disablePasswordAuth"`
	Groups                 []rbactypes.Group    `json:"groups,omitempty" yaml:"groups,omitempty"`
	IngressConfig          ingresstypes.Config  `json:"ingressConfig,omitempty" yaml:"ingressConfig,omitempty"`
	AdminConsoleAddress    string               `json:"adminConsoleAddress,omitempty" yaml:"adminConsoleAddress,omitempty"`
	IdentityServiceAddress string               `json:"identityServiceAddress,omitempty" yaml:"identityServiceAddress,omitempty"`
	DexConnectors          []dextypes.Connector `json:"dexConnectors,omitempty" yaml:"dexConnectors,omitempty"`
}

func (c *Config) Validate(ingressConfig ingresstypes.Config) error {
	if c.AdminConsoleAddress == "" && !ingressConfig.Enabled {
		return errors.New("Admin Console address must be provided via 'adminConsoleAddress' field, or by enabling ingress for the Admin Console")
	}

	if c.IdentityServiceAddress == "" && !c.IngressConfig.Enabled {
		return errors.New("Identity Service address must be provided via 'identityServiceAddress' field, or by enabling ingress for the Identity Service via 'ingressConfig' field")
	}

	return nil
}
