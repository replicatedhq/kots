package identity

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/ingress"
	ingresstypes "github.com/replicatedhq/kots/pkg/ingress/types"
)

var (
	KotsIdentityLabelKey   = "kots.io/identity"
	KotsIdentityLabelValue = "true"
)

func DexIssuerURL(ingressConfig ingresstypes.Config) string {
	return fmt.Sprintf("%s/dex", ingress.GetAddress(ingressConfig))
}

func DexCallbackURL(ingressConfig ingresstypes.Config) string {
	return fmt.Sprintf("%s/callback", DexIssuerURL(ingressConfig))
}
