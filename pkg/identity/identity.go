package identity

import (
	"fmt"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/rbac"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	WildcardGroupID = "*"
)

var (
	KotsIdentityLabelKey   = "kots.io/identity"
	KotsIdentityLabelValue = "true"

	DefaultGroups = []kotsv1beta1.IdentityGroup{DefaultGroup}
	DefaultGroup  = kotsv1beta1.IdentityGroup{
		ID:      WildcardGroupID,
		RoleIDs: []string{rbac.ClusterAdminRole.ID},
	}
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

func DexIssuerURL(identitySpec kotsv1beta1.IdentityConfigSpec) string {
	if identitySpec.IdentityServiceAddress != "" {
		return identitySpec.IdentityServiceAddress
	}
	return fmt.Sprintf("%s/dex", ingress.GetAddress(identitySpec.IngressConfig))
}

func DexCallbackURL(identitySpec kotsv1beta1.IdentityConfigSpec) string {
	return fmt.Sprintf("%s/callback", DexIssuerURL(identitySpec))
}
