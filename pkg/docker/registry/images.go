package registry

import (
	"strings"
)

func MakeProxiedImageURL(proxyHost string, appSlug string, image string) string {
	parts := strings.Split(image, "@")
	if len(parts) == 2 {
		return strings.Join([]string{proxyHost, "proxy", appSlug, parts[0]}, "/")
	}

	// TODO: host with a port breaks this
	parts = strings.Split(image, ":")
	return strings.Join([]string{proxyHost, "proxy", appSlug, parts[0]}, "/")
}
