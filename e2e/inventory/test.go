package inventory

import "github.com/replicatedhq/kots/e2e/kubectl"

type TestimParams map[string]interface{}

type Test struct {
	ID                     string // must match directory name in e2e/playwright/tests
	Name                   string // must match test-focus in .github/workflows/build-test.yaml
	TestimSuite            string
	TestimLabel            string
	Namespace              string
	AppSlug                string
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
