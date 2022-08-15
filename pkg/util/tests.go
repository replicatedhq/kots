package util

func TestGetenv(key string) string {
	switch key {
	case "POD_NAMESPACE":
		return PodNamespace
	default:
		return ""
	}
}
