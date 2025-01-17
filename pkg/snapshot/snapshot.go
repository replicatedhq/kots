package snapshot

import (
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	velerov1.AddToScheme(scheme.Scheme)
}
