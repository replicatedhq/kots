package inventory

import "github.com/replicatedhq/kots/e2e/kubectl"

type Test struct {
	Name            string
	Suite           string
	Label           string
	Namespace       string
	UpstreamURI     string
	UseMinimalRBAC  bool
	NeedsSnapshots  bool
	NeedsMonitoring bool
	NeedsRegistry   bool
	Setup           func(kubectlCLI *kubectl.CLI)
}
