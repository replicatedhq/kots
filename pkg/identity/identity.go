package identity

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/rbac"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
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

func DexIssuerURL(identitySpec kotsv1beta1.IdentitySpec) string {
	if identitySpec.IdentityServiceAddress != "" {
		return fmt.Sprintf("%s/dex", identitySpec.IdentityServiceAddress)
	}
	return fmt.Sprintf("%s/dex", ingress.GetAddress(identitySpec.IngressConfig))
}

func DexCallbackURL(identitySpec kotsv1beta1.IdentitySpec) string {
	return fmt.Sprintf("%s/callback", DexIssuerURL(identitySpec))
}

func EncodeSpec(spec kotsv1beta1.Identity) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	err := s.Encode(&spec, buf)
	return buf.Bytes(), err
}

func DecodeSpec(data []byte) (*kotsv1beta1.Identity, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, _, err := decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	spec, ok := decoded.(*kotsv1beta1.Identity)
	if !ok {
		return nil, errors.Errorf("wrong type %T", spec)
	}
	return spec, nil
}

func ConfigValidate(identitySpec kotsv1beta1.IdentitySpec, ingressSpec kotsv1beta1.IngressSpec) error {
	if identitySpec.AdminConsoleAddress == "" && (!ingressSpec.Enabled || ingressSpec.Ingress == nil) {
		return errors.New("adminConsoleAddress required or KOTS Admin Console ingress must be enabled")
	}

	if identitySpec.IdentityServiceAddress == "" && (!identitySpec.IngressConfig.Enabled || identitySpec.IngressConfig.Ingress == nil) {
		return errors.New("identityServiceAddress required or ingressConfig.ingress must be enabled")
	}

	return nil
}
