package cluster

import (
	"fmt"
	"io/ioutil"
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

	registriesConfig, err := k3dWriteRegistriesConfig(c.workspace, RegistryClusterIP)
	Expect(err).WithOffset(1).Should(Succeed(), "write registries config")

	session, err := k3dClusterCreate(c.clusterName, registriesConfig)
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
	GinkgoWriter.Printf("To set kubecontext run:\n  export KUBECONFIG=\"$(k3d kubeconfig merge %s)\"\n", c.GetClusterName())
	GinkgoWriter.Printf("To delete cluster run:\n  k3d cluster delete %s\n", c.GetClusterName())
}

func (c *K3d) GetKubeconfig() string {
	return c.kubeconfig
}

func (c *K3d) GetClusterName() string {
	return c.clusterName
}

func k3dClusterCreate(clusterName, registriesConfig string) (*gexec.Session, error) {
	return util.RunCommand(exec.Command(
		"k3d",
		"cluster",
		"create",
		"--kubeconfig-update-default=false",
		fmt.Sprintf("--registry-config=%s", registriesConfig),
		clusterName,
	))
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
	return util.RunCommand(cmd)
}

func k3dWriteKubeconfig(clusterName, kubeconfig string) (*gexec.Session, error) {
	return util.RunCommand(exec.Command(
		"k3d",
		"kubeconfig",
		"write",
		"--overwrite",
		fmt.Sprintf("--output=%s", kubeconfig),
		clusterName,
	))
}

func k3dWriteRegistriesConfig(workspace, clusterIP string) (string, error) {
	fileName := filepath.Join(workspace, "registries.yaml")
	fileContents := fmt.Sprintf(`mirrors:
  "%s:5000":
    endpoint:
    - "http://%s:5000"
`,
		clusterIP, clusterIP,
	)
	err := ioutil.WriteFile(fileName, []byte(fileContents), 0644)
	return fileName, err
}
