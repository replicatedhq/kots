package util

import "os"

var (
	PodNamespace     string = os.Getenv("POD_NAMESPACE")
	KotsadmNamespace string = os.Getenv("KOTSADM_TARGET_NAMESPACE")
)

func AppNamespace() string {
	if KotsadmNamespace != "" {
		return KotsadmNamespace
	}

	return PodNamespace
}
