package helm

import (
	"fmt"
	"path/filepath"

	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/util"
)

type CLI struct {
	workspace string
}

func NewCLI(workspace string) *CLI {
	return &CLI{
		workspace: workspace,
	}
}

func (c *CLI) RepoAdd(name, url string) (*gexec.Session, error) {
	return util.RunCommand(
		"helm",
		c.AppendCommonFlags(
			"repo",
			"add",
			name,
			url,
		)...,
	)
}

func (c *CLI) Install(kubeconfig string, args ...string) (*gexec.Session, error) {
	args = append(
		[]string{
			fmt.Sprintf("--kubeconfig=%s", kubeconfig),
			"install",
		},
		args...,
	)
	return util.RunCommand(
		"helm",
		c.AppendCommonFlags(args...)...,
	)
}

func (c *CLI) AppendCommonFlags(args ...string) []string {
	return append(
		[]string{
			fmt.Sprintf("--registry-config=%s", filepath.Join(c.workspace, "registry.json")),
			fmt.Sprintf("--repository-cache=%s", filepath.Join(c.workspace, "repository")),
			fmt.Sprintf("--repository-config=%s", filepath.Join(c.workspace, "repositories.yaml")),
		},
		args...,
	)
}
