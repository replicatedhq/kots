package testim

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/kots/e2e/inventory"
	"github.com/replicatedhq/kots/e2e/util"
)

type Client struct {
	AccessToken string
	Project     string
	Grid        string
	Branch      string
}

type RunOptions struct {
	TunnelPort string
	BaseUrl    string
	Params     map[string]interface{}
}

func NewClient(accessToken, project, grid, branch string) *Client {
	return &Client{
		AccessToken: accessToken,
		Project:     project,
		Grid:        grid,
		Branch:      branch,
	}
}

func (t *Client) NewRun(kubeconfig string, test inventory.Test, runOptions RunOptions) *Run {
	args := []string{
		fmt.Sprintf("--token=%s", t.AccessToken),
		fmt.Sprintf("--project=%s", t.Project),
		fmt.Sprintf("--grid=%s", t.Grid),
		fmt.Sprintf("--branch=%s", t.Branch),
		"--timeout=3600000",
	}

	params := map[string]interface{}{
		// skips snapshots volume assertions, velero will not backup rancher/local-path-provisioner volumes
		"testDisableSnapshotsVolumeAssertions": true,
	}
	for k, v := range runOptions.Params {
		params[k] = v
	}
	paramsJson, err := json.Marshal(params)
	Expect(err).WithOffset(1).Should(Succeed(), "Create testim params")
	args = append(args, fmt.Sprintf(`--params=%s`, paramsJson))

	if test.TestimSuite != "" {
		args = append(
			args,
			fmt.Sprintf("--suite=%s", test.TestimSuite),
		)
	}
	if test.TestimLabel != "" {
		args = append(
			args,
			fmt.Sprintf("--label=%s", test.TestimLabel),
		)
	}
	if test.Browser != "" {
		args = append(
			args,
			fmt.Sprintf("--browser=%s", test.Browser),
			"--mode=selenium",
		)
	}
	if runOptions.BaseUrl != "" {
		args = append(
			args,
			fmt.Sprintf("--base-url=%s", runOptions.BaseUrl),
		)
	}
	if runOptions.TunnelPort != "" {
		args = append(
			args,
			"--tunnel",
			fmt.Sprintf("--tunnel-port=%s", runOptions.TunnelPort),
		)
	}
	cmd := exec.Command("testim", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	cmd.Env = append(cmd.Env, "NODE_OPTIONS=--max-old-space-size=4096")
	session, err := util.RunCommand(cmd)
	Expect(err).WithOffset(1).Should(Succeed(), "Run testim tests failed")
	return &Run{session}
}
