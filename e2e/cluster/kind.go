package cluster

import (
	"path/filepath"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/ginkgo/v2"
	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"sigs.k8s.io/kind/pkg/cluster"
)

type Kind struct {
	workspace    string
	kubeconfig   string
	clusterName  string
	kindProvider *cluster.Provider
}

func NewKind(workspace string) *Kind {
	c := &Kind{
		workspace: workspace,
	}

	c.kubeconfig = filepath.Join(c.workspace, ".kubeconfig")
	c.clusterName = filepath.Base(c.workspace)

	logger := NewLogger(GinkgoWriter, 0)
	c.kindProvider = cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
	)
	err := c.kindProvider.Create(
		c.clusterName,
		cluster.CreateWithWaitForReady(time.Minute),
		cluster.CreateWithKubeconfigPath(c.kubeconfig),
	)
	Expect(err).WithOffset(1).Should(Succeed(), "create cluster")
	return c
}

func (c *Kind) Teardown() {
	if c.kindProvider != nil && c.clusterName != "" && c.kubeconfig != "" {
		err := c.kindProvider.Delete(c.clusterName, c.kubeconfig)
		Expect(err).WithOffset(1).Should(Succeed(), "delete cluster")
	}
}

func (c *Kind) PrintDebugInfo() {
	GinkgoWriter.Printf("To set kubecontext run:\n  export KUBECONFIG=%s\n", c.GetKubeconfig())
	GinkgoWriter.Printf("To delete cluster run:\n  kind delete cluster --name=%s\n", c.GetClusterName())
}

func (c *Kind) GetKubeconfig() string {
	return c.kubeconfig
}

func (c *Kind) GetClusterName() string {
	return c.clusterName
}
