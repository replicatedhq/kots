package testim

import (
	"fmt"
	"os"
	"os/exec"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/kots/e2e/testim/inventory"
	"github.com/replicatedhq/kots/e2e/util"
)

type Client struct {
	AccessToken string
	Project     string
	Grid        string
	Branch      string
}

func NewClient(accessToken, project, grid, branch string) *Client {
	return &Client{
		AccessToken: accessToken,
		Project:     project,
		Grid:        grid,
		Branch:      branch,
	}
}

func (t *Client) NewRun(kubeconfig string, test inventory.Test, adminConsolePort string) *Run {
	args := []string{
		fmt.Sprintf("--token=%s", t.AccessToken),
		fmt.Sprintf("--project=%s", t.Project),
		fmt.Sprintf("--grid=%s", t.Grid),
		fmt.Sprintf("--branch=%s", t.Branch),
		"--timeout=3600000",
	}
	if test.Suite != "" {
		args = append(
			args,
			fmt.Sprintf("--suite=%s", test.Suite),
		)
	}
	if test.Label != "" {
		args = append(
			args,
			fmt.Sprintf("--label=%s", test.Label),
		)
	}
	if adminConsolePort != "" {
		args = append(
			args,
			"--tunnel",
			fmt.Sprintf("--tunnel-port=%s", adminConsolePort),
		)
	}
	cmd := exec.Command("testim", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	session, err := util.RunCommand(cmd)
	Expect(err).WithOffset(1).Should(Succeed(), "Run testim tests failed")
	return &Run{session}
}
