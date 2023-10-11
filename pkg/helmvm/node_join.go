package helmvm

import (
	"time"

	"k8s.io/client-go/kubernetes"
)

// GenerateAddNodeCommand will generate the HelmVM node add command for a primary or secondary node
func GenerateAddNodeCommand(client kubernetes.Interface, primary bool) ([]string, *time.Time, error) {
	return nil, nil, nil
}
