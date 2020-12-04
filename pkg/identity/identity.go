package identity

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc"
	dexoidc "github.com/dexidp/dex/connector/oidc"
	"github.com/pkg/errors"
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

func ConfigValidate(ctx context.Context, namespace string, identitySpec kotsv1beta1.IdentityConfigSpec, ingressSpec kotsv1beta1.IngressConfigSpec) error {
	if identitySpec.AdminConsoleAddress == "" && (!ingressSpec.Enabled || ingressSpec.Ingress == nil) {
		return errors.New("adminConsoleAddress required or KOTS Admin Console ingress must be enabled")
	}

	if identitySpec.IdentityServiceAddress == "" && (!identitySpec.IngressConfig.Enabled || identitySpec.IngressConfig.Ingress == nil) {
		return errors.New("identityServiceAddress required or ingressConfig.ingress must be enabled")
	}

	if err := evaluateDexConnectorsValue(ctx, namespace, &identitySpec.DexConnectors); err != nil {
		return errors.Wrap(err, "failed to evaluate dex connectors value")
	}

	// validate issuers
	conns, err := IdentityDexConnectorsToDexTypeConnectors(identitySpec.DexConnectors.Value)
	if err != nil {
		return errors.Wrap(err, "failed to map identity dex connectors to dex type connectors")
	}
	for _, conn := range conns {
		switch c := conn.Config.(type) {
		case *dexoidc.Config:
			httpClient, err := HTTPClient(ctx, namespace)
			if err != nil {
				return errors.Wrap(err, "failed to init http client")
			}

			oidcClientCtx := oidc.ClientContext(ctx, httpClient)
			_, err = oidc.NewProvider(oidcClientCtx, c.Issuer)
			if err != nil {
				return errors.Wrapf(err, "failed to query provider %q", c.Issuer)
			}
		}
	}

	return nil
}
