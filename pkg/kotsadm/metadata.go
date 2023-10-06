package kotsadm

import (
	"github.com/replicatedhq/kots/pkg/helmvm"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kurl"
	"k8s.io/client-go/kubernetes"
)

func GetMetadata(clientset kubernetes.Interface) types.Metadata {
	isKurl, _ := kurl.IsKurl(clientset)
	isHelmVM, _ := helmvm.IsHelmVM(clientset)

	metadata := types.Metadata{
		IsAirgap: IsAirgap(),
		IsKurl:   isKurl,
		IsHelmVM: isHelmVM,
	}

	return metadata
}
