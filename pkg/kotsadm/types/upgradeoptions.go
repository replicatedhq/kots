package types

import (
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type UpgradeOptions struct {
	Namespace             string
	KubernetesConfigFlags *genericclioptions.ConfigFlags
	ForceUpgradeKurl      bool
	Timeout               time.Duration

	KotsadmOptions KotsadmOptions
}
