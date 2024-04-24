package playwright

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/inventory"
	"github.com/replicatedhq/kots/e2e/util"
)

type Client struct {
}

type Run struct {
	session *gexec.Session
}

type RunOptions struct {
	Port string
}

func NewClient() *Client {
	return &Client{}
}

func (t *Client) HasTest(test inventory.Test) bool {
	if test.ID == "" {
		return false
	}
	_, err := os.Stat(fmt.Sprintf("/playwright/tests/%s", test.ID))
	return err == nil
}

func (t *Client) NewRun(kubeconfig string, test inventory.Test, runOptions RunOptions) *Run {
	args := []string{
		"playwright",
		"test",
		test.ID,
	}

	cmd := exec.Command("npx", args...)
	cmd.Dir = "/playwright"
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%s", runOptions.Port))
	cmd.Env = append(cmd.Env, fmt.Sprintf("NAMESPACE=%s", test.Namespace))
	cmd.Env = append(cmd.Env, fmt.Sprintf("APP_SLUG=%s", test.AppSlug))
	cmd.Env = append(cmd.Env, fmt.Sprintf("TEST_PATH=%s", filepath.Join("tests", test.ID)))
	cmd.Env = append(cmd.Env, "NODE_OPTIONS=--max-old-space-size=4096")
	session, err := util.RunCommand(cmd)
	Expect(err).WithOffset(1).Should(Succeed(), "Run playwright test failed")
	return &Run{session}
}

func (r *Run) ShouldSucceed() {
	Eventually(r.session).WithOffset(1).WithTimeout(30*time.Minute).Should(gexec.Exit(), "Run playwright test timed out")
	Expect(r.session.ExitCode()).Should(Equal(0), "Run playwright test failed with non-zero exit code")
}
