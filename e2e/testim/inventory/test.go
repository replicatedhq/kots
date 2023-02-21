package inventory

import "github.com/replicatedhq/kots/e2e/kubectl"

type TestimParams map[string]interface{}

type Test struct {
	Name                   string // must match test-focus in .github/workflows/build-test.yaml
	Suite                  string
	Label                  string
	Namespace              string
	UpstreamURI            string
	Browser                string
	UseMinimalRBAC         bool
	SkipCompatibilityCheck bool
	NeedsSnapshots         bool
	NeedsMonitoring        bool
	NeedsRegistry          bool
	IsHelmManaged          bool
	Setup                  func(kubectlCLI *kubectl.CLI) TestimParams
}
