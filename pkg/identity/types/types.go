package types

import "fmt"

func DeploymentName(prefix string) string {
	return prefixName(prefix, "dex")
}

func ServiceName(prefix string) string {
	return prefixName(prefix, "dex")
}

func ServicePort() int32 {
	return 5556
}

func prefixName(prefix, name string) string {
	return fmt.Sprintf("%s-%s", prefix, name)
}
