package kotsadm

import (
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kurl"
)

func GetMetadata() types.Metadata {
	// expected to fail for minimal rbac
	isKurl, _ := kurl.IsKurl()
	metadata := types.Metadata{
		IsAirgap: IsAirgap(),
		IsKurl:   isKurl,
	}

	return metadata
}
