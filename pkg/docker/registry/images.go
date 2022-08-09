package registry

import (
	"strings"
)

func MakeProxiedImageURL(proxyHost string, appSlug string, image string) string {
	parts := strings.Split(image, "@")
	if len(parts) == 2 {
		// we have a digest, but need to also check for a tag
		parts = strings.Split(parts[0], ":")
		return strings.Join([]string{proxyHost, "proxy", appSlug, parts[0]}, "/")
	}

	// TODO: host with a port breaks this
	parts = strings.Split(image, ":")
	return strings.Join([]string{proxyHost, "proxy", appSlug, parts[0]}, "/")
}
