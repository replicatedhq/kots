package ingress

import (
	"bytes"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

func EncodeSpec(spec kotsv1beta1.Ingress) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	err := s.Encode(&spec, buf)
	return buf.Bytes(), err
}

func DecodeSpec(data []byte) (*kotsv1beta1.Ingress, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, _, err := decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	spec, ok := decoded.(*kotsv1beta1.Ingress)
	if !ok {
		return nil, errors.Errorf("wrong type %T", spec)
	}
	return spec, nil
}

func GetAddress(ingressSpec kotsv1beta1.IngressSpec) string {
	switch {
	case ingressSpec.Ingress != nil:
		return getIngressConfigAddress(*ingressSpec.Ingress)

	case ingressSpec.NodePort != nil:
		return "" // TODO
	}

	return ""
}

func getIngressConfigAddress(ingressConfig kotsv1beta1.IngressConfig) string {
	var u url.URL
	if ingressConfig.TLSSecretName != "" {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}

	u.Host = ingressConfig.Host
	u.Path = ingressConfig.Path

	return strings.TrimRight(u.String(), "/")
}
