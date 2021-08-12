package util

import "os"

var (
	PodNamespace           string = os.Getenv("POD_NAMESPACE")
	KotsadmTargetNamespace string = os.Getenv("KOTSADM_TARGET_NAMESPACE")
)

func AppNamespace() string {
	if KotsadmTargetNamespace != "" {
		return KotsadmTargetNamespace
	}

	return PodNamespace
}
