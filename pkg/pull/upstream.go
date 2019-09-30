package pull

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/util"
)

func RewriteUpstream(upstreamURI string) string {
	if !util.IsURL(upstreamURI) {
		upstreamURI = fmt.Sprintf("replicated://%s", upstreamURI)
	}

	return upstreamURI
}
