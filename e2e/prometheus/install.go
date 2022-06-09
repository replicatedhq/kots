package prometheus

import (
	"fmt"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/helm"
)

const (
	DefaultNamespace   = "monitoring"
	DefaultReleaseName = "k8s"
)

type Prometheus struct {
	options Options
}

type Options struct {
	Namespace   string
	ReleaseName string
}

func New(opts Options) Prometheus {
	p := Prometheus{
		Options{
			Namespace:   opts.Namespace,
			ReleaseName: opts.ReleaseName,
		},
	}
	if p.options.Namespace == "" {
		p.options.Namespace = DefaultNamespace
	}
	if p.options.ReleaseName == "" {
		p.options.ReleaseName = DefaultReleaseName
	}
	return p
}

func (m *Prometheus) Install(helmCLI *helm.CLI, kubeconfig string) {
	session, err := helmCLI.RepoAdd("prometheus-community", "https://prometheus-community.github.io/helm-charts")
	Expect(err).WithOffset(1).Should(Succeed(), "helm repo add")
	Eventually(session).WithOffset(1).WithTimeout(time.Minute).Should(gexec.Exit(0), "helm repo add")

	session, err = helmCLI.Install(
		kubeconfig,
		"--create-namespace",
		fmt.Sprintf("--namespace=%s", m.options.Namespace),
		"--wait",
		"--set=server.fullnameOverride=prometheus-k8s",
		"--set=server.service.servicePort=9090",
		m.options.ReleaseName,
		"prometheus-community/prometheus",
	)
	Expect(err).WithOffset(1).Should(Succeed(), "helm install")
	Eventually(session).WithOffset(1).WithTimeout(2*time.Minute).Should(gexec.Exit(0), "helm install")
}

func (m *Prometheus) GetURL() string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:9000", m.options.ReleaseName, m.options.Namespace)
}
