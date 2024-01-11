package registry

import (
	"strings"

	"github.com/replicatedhq/kots/pkg/imageutil"
)

func MakeProxiedImageURL(proxyHost string, appSlug string, image string) string {
	untagged := imageutil.StripImageTagAndDigest(image)

	registryHost := strings.Split(untagged, "/")[0]
	if registryHost == proxyHost {
		// already proxied
		return untagged
	}

	return strings.Join([]string{proxyHost, "proxy", appSlug, untagged}, "/")
}
