package kubectl

import (
	"context"
	"fmt"
	"os/exec"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/ginkgo/v2"
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

func (c *CLI) Command(ctx context.Context, args ...string) *exec.Cmd {
	args = append(
		[]string{
			fmt.Sprintf("--cache-dir=%s/.kube/cache", c.workspace),
			fmt.Sprintf("--kubeconfig=%s", c.kubeconfig),
		},
		args...,
	)
	return exec.CommandContext(ctx, "kubectl", args...)
}

func (c *CLI) RunCommand(ctx context.Context, args ...string) ([]byte, error) {
	return c.Command(ctx, args...).CombinedOutput()
}

func (c *CLI) GetPods(ctx context.Context, namespace string) {
	out, _ := c.RunCommand(ctx, "get", "pods", fmt.Sprintf("--namespace=%s", namespace))
	GinkgoWriter.Write(out)
}

func (c *CLI) GetAllPods(ctx context.Context) {
	out, _ := c.RunCommand(ctx, "get", "pods", "--all-namespaces")
	GinkgoWriter.Write(out)
}

func (c *CLI) DescribeNodes(ctx context.Context) {
	out, _ := c.RunCommand(ctx, "describe", "nodes")
	GinkgoWriter.Write(out)
}
