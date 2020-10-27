package types

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type EnableDisasterRecoveryOptions struct {
	Namespace             string
	KubernetesConfigFlags *genericclioptions.ConfigFlags
}
