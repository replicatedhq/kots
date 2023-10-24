package kotsadm

import (
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kurl"
	"k8s.io/client-go/kubernetes"
)

func GetMetadata(clientset kubernetes.Interface) types.Metadata {
	isKurl, _ := kurl.IsKurl(clientset)
	isEmbeddedCluster, _ := embeddedcluster.IsEmbeddedCluster(clientset)

	metadata := types.Metadata{
		IsAirgap:          IsAirgap(),
		IsKurl:            isKurl,
		IsEmbeddedCluster: isEmbeddedCluster,
	}

	return metadata
}
