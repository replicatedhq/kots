package kubectl

import (
	"fmt"
	"os/exec"

	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/util"
)

type CLI struct {
	workspace  string
	kubeconfig string
}

func NewCLI(workspace, kubeconfig string) *CLI {
	return &CLI{
		workspace:  workspace,
		kubeconfig: kubeconfig,
	}
}

func (c *CLI) RunCommand(args ...string) (*gexec.Session, error) {
	args = append(
		[]string{
			fmt.Sprintf("--cache-dir=%s/.kube/cache", c.workspace),
			fmt.Sprintf("--kubeconfig=%s", c.kubeconfig),
		},
		args...,
	)
	return util.RunCommand(exec.Command("kubectl", args...))
}

func (c *CLI) GetPods(namespace string) {
	session, _ := c.RunCommand("get", "pods", fmt.Sprintf("--namespace=%s", namespace))
	session.Wait()
}

func (c *CLI) GetAllPods() {
	session, _ := c.RunCommand("get", "pods", "--all-namespaces")
	session.Wait()
}

func (c *CLI) DescribeNodes() {
	session, _ := c.RunCommand("describe", "nodes")
	session.Wait()
}
