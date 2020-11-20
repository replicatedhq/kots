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
		return strings.TrimRight(config.NodePort.Address, "/")

	case config.External != nil:
		return strings.TrimRight(config.External.Address, "/")
	}

	return ""
}

func getIngressConfigAddress(ingressConfig types.IngressConfig) string {
	if ingressConfig.Address != "" {
		return strings.TrimRight(ingressConfig.Address, "/")
	}

	var u url.URL
	if len(ingressConfig.TLS) > 0 {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}

	u.Host = ingressConfig.Host
	u.Path = ingressConfig.Path

	return strings.TrimRight(u.String(), "/")
}
