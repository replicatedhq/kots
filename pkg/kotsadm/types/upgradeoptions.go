package types

import "k8s.io/cli-runtime/pkg/genericclioptions"

type UpgradeOptions struct {
	Namespace             string
	KubernetesConfigFlags *genericclioptions.ConfigFlags
}
