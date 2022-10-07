package types

import "fmt"

func Namespace(prefix string) string {
	return prefixName(prefix, "dex")
}

func ClusterRoleName(prefix string) string {
	return prefixName(prefix, "dex")
}

func ServiceAccountName(prefix string) string {
	return prefixName(prefix, "dex")
}

func ClusterRoleBindingName(prefix string) string {
	return prefixName(prefix, "dex")
}

func DeploymentName(prefix string) string {
	return prefixName(prefix, "dex")
}

func SecretName(prefix string) string {
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
