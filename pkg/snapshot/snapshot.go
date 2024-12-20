package snapshot

import (
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	veleroscheme.AddToScheme(scheme.Scheme)
}
