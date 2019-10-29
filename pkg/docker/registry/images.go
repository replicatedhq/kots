package registry

import (
	"strings"
)

func MakeProxiedImageURL(proxyHost string, appSlug string, image string) string {
	// TODO: host with a port breaks this
	parts := strings.Split(image, ":")
	return strings.Join([]string{proxyHost, "proxy", appSlug, parts[0]}, "/")
}
