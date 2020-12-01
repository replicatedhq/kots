package ingress

import (
	"net/url"
	"strings"

	"github.com/replicatedhq/kots/pkg/ingress/types"
)

func GetAddress(config types.Config) string {
	switch {
	case config.Ingress != nil:
		return getIngressConfigAddress(*config.Ingress)

	case config.NodePort != nil:
		return "" // TODO
	}

	return ""
}

func getIngressConfigAddress(ingressConfig types.IngressConfig) string {
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
