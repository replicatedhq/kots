package kotsadm

import (
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/util"
	"k8s.io/client-go/kubernetes"
)

func GetMetadata(clientset kubernetes.Interface) types.Metadata {
	isKurl, _ := kurl.IsKurl(clientset)

	metadata := types.Metadata{
		IsAirgap:          IsAirgap(),
		IsKurl:            isKurl,
		IsEmbeddedCluster: util.IsEmbeddedCluster(),
	}

	return metadata
}
