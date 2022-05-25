package testim

import (
	"fmt"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
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

func (t *Client) Run(test inventory.Test, adminConsolePort string) {
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
	session, err := util.RunCommand("testim", args...)
	Expect(err).WithOffset(1).Should(Succeed(), "run testim tests")
	Eventually(session).WithOffset(1).WithTimeout(30 * time.Minute).Should(gexec.Exit(0))
}
