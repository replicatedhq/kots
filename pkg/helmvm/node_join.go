package helmvm

import (
	"time"

	"k8s.io/client-go/kubernetes"
)

// GenerateAddNodeCommand will generate the HelmVM node add command for a primary or secondary node
func GenerateAddNodeCommand(client kubernetes.Interface, primary bool) ([]string, *time.Time, error) {
	tomorrow := time.Now().Add(time.Hour * 24)
	if primary {
		return []string{"this is a primary join command string", "that can be multiple strings"}, &tomorrow, nil
	} else {
		return []string{"this is a secondary join command string", "that can be multiple strings"}, &tomorrow, nil
	}
}
