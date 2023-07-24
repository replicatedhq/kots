package ingress

import (
	"net/url"
	"strings"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func GetAddress(ingressSpec kotsv1beta1.IngressConfigSpec) string {
	switch {
	case ingressSpec.Ingress != nil:
		return getIngressConfigAddress(*ingressSpec.Ingress)

	case ingressSpec.NodePort != nil:
		return "" // TODO
	}

	return ""
}

func getIngressConfigAddress(ingressConfig kotsv1beta1.IngressResourceConfig) string {
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
