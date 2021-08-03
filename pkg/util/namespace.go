package util

var (
	PodNamespace     string
	KotsadmNamespace string
)

func AppNamespace() string {
	if KotsadmNamespace == "" {
		return KotsadmNamespace
	}

	return PodNamespace
}
