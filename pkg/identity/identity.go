package identity

import (
	"fmt"

	types "github.com/replicatedhq/kots/pkg/identity/types"
	"github.com/replicatedhq/kots/pkg/ingress"
)

var (
	KotsIdentityLabelKey   = "kots.io/identity"
	KotsIdentityLabelValue = "true"
)

func DexIssuerURL(identityConfig types.Config) string {
	if identityConfig.IdentityServiceAddress != "" {
		return fmt.Sprintf("%s/dex", identityConfig.IdentityServiceAddress)
	}
	return fmt.Sprintf("%s/dex", ingress.GetAddress(identityConfig.IngressConfig))
}

func DexCallbackURL(identityConfig types.Config) string {
	return fmt.Sprintf("%s/callback", DexIssuerURL(identityConfig))
}
