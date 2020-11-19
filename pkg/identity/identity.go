package identity

import (
	"fmt"
	"path"

	"github.com/replicatedhq/kots/pkg/identity/types"
)

var (
	KotsIdentityLabelKey   = "kots.io/identity"
	KotsIdentityLabelValue = "true"
)

func DexIssuerURL(identityConfig types.Config) string {
	return fmt.Sprintf("http://%s", path.Join(identityConfig.IngressConfig.Host, identityConfig.IngressPath(), "dex"))
}

func DexCallbackURL(identityConfig types.Config) string {
	return fmt.Sprintf("%s/callback", DexIssuerURL(identityConfig))
}
