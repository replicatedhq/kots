package identity

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/rbac"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	WildcardGroupID = "*"
)

var (
	DefaultGroups = []kotsv1beta1.IdentityConfigGroup{DefaultGroup}
	DefaultGroup  = kotsv1beta1.IdentityConfigGroup{
		ID:      WildcardGroupID,
		RoleIDs: []string{rbac.ClusterAdminRole.ID},
	}
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

func DexIssuerURL(identityConfigSpec kotsv1beta1.IdentityConfigSpec) string {
	if identityConfigSpec.IdentityServiceAddress != "" {
		return identityConfigSpec.IdentityServiceAddress
	}
	return fmt.Sprintf("%s/dex", ingress.GetAddress(identityConfigSpec.IngressConfig))
}

func DexCallbackURL(identityConfigSpec kotsv1beta1.IdentityConfigSpec) string {
	return fmt.Sprintf("%s/callback", DexIssuerURL(identityConfigSpec))
}
