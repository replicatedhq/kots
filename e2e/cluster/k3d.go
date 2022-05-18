package cluster

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/ginkgo/v2"
	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/util"
)

type K3d struct {
	workspace   string
	kubeconfig  string
	clusterName string
}

func NewK3d(workspace string) *K3d {
	c := &K3d{
		workspace: workspace,
	}

	c.kubeconfig = filepath.Join(c.workspace, ".kubeconfig")
	c.clusterName = filepath.Base(c.workspace)

	session, err := k3dClusterCreate(c.clusterName)
	if err != nil {
		c.Teardown()
	}
	Expect(err).WithOffset(1).Should(Succeed(), "create cluster")
	Eventually(session).WithOffset(1).WithTimeout(time.Minute).Should(gexec.Exit(0), "create cluster")

	session, err = k3dWriteKubeconfig(c.clusterName, c.kubeconfig)
	if err != nil {
		c.Teardown()
	}
	Expect(err).WithOffset(1).Should(Succeed(), "write kubeconfig")
	Eventually(session).WithOffset(1).WithTimeout(time.Minute).Should(gexec.Exit(0), "write kubeconfig")
	return c
}

func (c *K3d) Teardown() {
	if c.clusterName != "" {
		session, err := k3dClusterDelete(c.clusterName, c.kubeconfig)
		Expect(err).WithOffset(1).Should(Succeed(), "delete cluster")
		Eventually(session).WithOffset(1).WithTimeout(time.Minute).Should(gexec.Exit(0), "delete cluster")
	}
}

func (c *K3d) PrintDebugInfo() {
	GinkgoWriter.Printf("To set kubecontext run:\n  export KUBECONFIG=%s\n", c.GetKubeconfig())
	GinkgoWriter.Printf("To delete cluster run:\n  k3d cluster delete %s\n", c.GetClusterName())
}

func (c *K3d) GetKubeconfig() string {
	return c.kubeconfig
}

func (c *K3d) GetClusterName() string {
	return c.clusterName
}

func k3dClusterCreate(clusterName string) (*gexec.Session, error) {
	return util.RunCommand(
		"k3d",
		"cluster",
		"create",
		"--kubeconfig-update-default=false",
		clusterName,
	)
}

func k3dClusterDelete(clusterName, kubeconfig string) (*gexec.Session, error) {
	cmd := exec.Command(
		"k3d",
		"cluster",
		"delete",
		clusterName,
	)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	return gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
}

func k3dWriteKubeconfig(clusterName, kubeconfig string) (*gexec.Session, error) {
	return util.RunCommand(
		"k3d",
		"kubeconfig",
		"write",
		"--overwrite",
		fmt.Sprintf("--output=%s", kubeconfig),
		clusterName,
	)
}
