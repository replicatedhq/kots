package kotsadm

import (
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kurl"
)

func GetMetadata() types.Metadata {
	metadata := types.Metadata{
		IsAirgap: IsAirgap(),
		IsKurl:   kurl.IsKurl(),
	}

	return metadata
}
