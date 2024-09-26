package inventory

import "github.com/replicatedhq/kots/e2e/kubectl"

type TestimParams map[string]interface{}

type Test struct {
	ID                     string // must match directory name in e2e/playwright/tests
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
	Setup                  func(kubectlCLI *kubectl.CLI) TestimParams
}
