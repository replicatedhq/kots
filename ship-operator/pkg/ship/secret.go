package ship

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// get a list of Shipwatch names from annotations
func GetShipWatchInstanceNamesFromMeta(meta metav1.Object) []string {
	value := meta.GetAnnotations()["shipwatch"]
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}

func HasSecretMeta(meta metav1.Object, instanceName string) bool {
	names := GetShipWatchInstanceNamesFromMeta(meta)
	for _, name := range names {
		if name == instanceName {
			return true
		}
	}
	return false
}
