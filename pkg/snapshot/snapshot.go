package snapshot

import (
	veleroscheme "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/scheme"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	veleroscheme.AddToScheme(scheme.Scheme)
}
