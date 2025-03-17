package inventory

import (
	"path/filepath"

	"github.com/replicatedhq/kots/e2e/kubectl"
)

type Test struct {
	ID                     string // must match directory name in e2e/playwright/tests
	dir                    string // defaults to "tests"
	Namespace              string
	AppSlug                string
	UpstreamURI            string
	Browser                string
	UseMinimalRBAC         bool
	SkipCompatibilityCheck bool
	NeedsSnapshots         bool
	NeedsMonitoring        bool
	NeedsRegistry          bool
	Setup                  func(kubectlCLI *kubectl.CLI)
	ExtraEnv               []string
}

func (t *Test) Dir() string {
	if t.dir == "" {
		return "tests"
	}
	return t.dir
}

func (t *Test) Path() string {
	return filepath.Join(t.Dir(), t.ID)
}
