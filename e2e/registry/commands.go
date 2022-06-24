package registry

import (
	"fmt"
	"time"

	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/helm"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
)

const (
	DefaultNamespace   = "registry"
	DefaultReleaseName = "registry"
)

type registry struct {
	helmCLI    *helm.CLI
	kubeconfig string
}

func New(helmCLI *helm.CLI, kubeconfig string) *registry {
	return &registry{
		helmCLI:    helmCLI,
		kubeconfig: kubeconfig,
	}
}

type Options struct {
	Namespace   string
	ReleaseName string
}

func (r *registry) Install(opts Options) {
	if opts.Namespace == "" {
		opts.Namespace = DefaultNamespace
	}
	if opts.ReleaseName == "" {
		opts.ReleaseName = DefaultReleaseName
	}
	session, err := r.helmCLI.RepoAdd("twuni", "https://helm.twun.io")
	Expect(err).WithOffset(1).Should(Succeed(), "helm repo add")
	Eventually(session).WithOffset(1).WithTimeout(time.Minute).Should(gexec.Exit(0), "helm repo add")

	session, err = r.helmCLI.Install(
		r.kubeconfig,
		"--create-namespace",
		fmt.Sprintf("--namespace=%s", opts.Namespace),
		"--wait",
		"--set=fullnameOverride=registry",
		opts.ReleaseName,
		"twuni/docker-registry",
	)
	Expect(err).WithOffset(1).Should(Succeed(), "helm install")
	Eventually(session).WithOffset(1).WithTimeout(2*time.Minute).Should(gexec.Exit(0), "helm install")
}
