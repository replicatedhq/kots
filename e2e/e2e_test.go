package e2e

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/ginkgo/v2"
	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/cluster"
	"github.com/replicatedhq/kots/e2e/helm"
	"github.com/replicatedhq/kots/e2e/kots"
	"github.com/replicatedhq/kots/e2e/kubectl"
	"github.com/replicatedhq/kots/e2e/minio"
	"github.com/replicatedhq/kots/e2e/prometheus"
	"github.com/replicatedhq/kots/e2e/registry"
	"github.com/replicatedhq/kots/e2e/testim"
	"github.com/replicatedhq/kots/e2e/testim/inventory"
	"github.com/replicatedhq/kots/e2e/util"
	"github.com/replicatedhq/kots/e2e/velero"
	"github.com/replicatedhq/kots/e2e/workspace"
)

var testimClient *testim.Client
var helmCLI *helm.CLI
var veleroCLI *velero.CLI
var kotsInstaller *kots.Installer

var (
	testimBranch          string
	testimBaseUrl         string
	skipTeardown          bool
	existingKubeconfig    string
	kotsadmImageRegistry  string
	kotsadmImageNamespace string
	kotsadmImageTag       string
	airgap                bool
	isOpenShift           bool
	kotsadmForwardPort    string
	kotsHelmChartURL      string
	kotsHelmChartVersion  string
	kotsDockerhubUsername string
	kotsDockerhubPassword string
)

func init() {
	flag.StringVar(&testimBranch, "testim-branch", "master", "testim branch to use")
	flag.StringVar(&testimBaseUrl, "testim-base-url", "", "override the base url that testim will use")
	flag.StringVar(&existingKubeconfig, "existing-kubeconfig", "", "use kubeconfig from existing cluster, do not create clusters (only for use with targeted testing)")
	flag.BoolVar(&skipTeardown, "skip-teardown", false, "do not tear down clusters")
	flag.StringVar(&kotsadmImageRegistry, "kotsadm-image-registry", "", "override the kotsadm images registry")
	flag.StringVar(&kotsadmImageNamespace, "kotsadm-image-namespace", "", "override the kotsadm images registry namespace")
	flag.StringVar(&kotsadmImageTag, "kotsadm-image-tag", "alpha", "override the kotsadm images tag")
	flag.BoolVar(&airgap, "airgap", false, "run install in airgapped mode")
	flag.BoolVar(&isOpenShift, "is-openshift", false, "the cluster is an openshift cluster")
	flag.StringVar(&kotsadmForwardPort, "kotsadm-forward-port", "", "sets the port that the admin console will be exposed on instead of generating a random one")
	flag.StringVar(&kotsHelmChartURL, "kots-helm-chart-url", "", "kots helm chart url")
	flag.StringVar(&kotsHelmChartVersion, "kots-helm-chart-version", "", "kots helm chart version")
	flag.StringVar(&kotsDockerhubUsername, "kots-dockerhub-username", "", "kots dockerhub username")
	flag.StringVar(&kotsDockerhubPassword, "kots-dockerhub-password", "", "kots dockerhub password")
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	testimAccessToken := os.Getenv("TESTIM_ACCESS_TOKEN")
	Expect(testimAccessToken).ShouldNot(BeEmpty(), "TESTIM_ACCESS_TOKEN required")

	Expect(util.CommandExists("kubectl")).To(BeTrue(), "kubectl required")
	Expect(util.CommandExists("helm")).To(BeTrue(), "helm required")
	Expect(util.CommandExists("velero")).To(BeTrue(), "velero required")
	Expect(util.CommandExists("testim")).To(BeTrue(), "testim required")
	Expect(util.CommandExists("kots")).To(BeTrue(), "kots required")

	w := workspace.New()
	DeferCleanup(w.Teardown)

	testimClient = testim.NewClient(
		testimAccessToken,
		util.EnvOrDefault("TESTIM_PROJECT_ID", "wpYAooUimFDgQxY73r17"),
		"Testim-grid",
		testimBranch,
	)

	helmCLI = helm.NewCLI(w.GetDir())

	veleroCLI = velero.NewCLI(w.GetDir(), isOpenShift)

	kotsInstaller = kots.NewInstaller(kotsadmImageRegistry, kotsadmImageNamespace, kotsadmImageTag, airgap, kotsDockerhubUsername, kotsDockerhubPassword)
})

var _ = AfterSuite(func() {
	gexec.KillAndWait()
})

var _ = Describe("E2E", func() {

	var w workspace.Workspace

	BeforeEach(func() {
		w = workspace.New()
		if !skipTeardown {
			DeferCleanup(w.Teardown)
		}
	})

	Context("with an online cluster", func() {

		var c cluster.Interface
		var kubectlCLI *kubectl.CLI
		var testimRun *testim.Run

		BeforeEach(func() {
			if existingKubeconfig != "" {
				c = cluster.NewExisting(existingKubeconfig)
			} else {
				k3d := cluster.NewK3d(w.GetDir())
				DeferCleanup(k3d.PrintDebugInfo)
				c = k3d
			}
			if !skipTeardown {
				DeferCleanup(c.Teardown)
			}

			kubectlCLI = kubectl.NewCLI(w.GetDir(), c.GetKubeconfig())
		})

		AfterEach(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Debug info
			GinkgoWriter.Println("\n")
			if kubectlCLI != nil {
				kubectlCLI.GetAllPods(ctx)
				kubectlCLI.DescribeNodes(ctx)
			}
			if testimRun != nil {
				testimRun.PrintDebugInfo()
			}
		})

		DescribeTable(
			"install kots and run the test",
			func(test inventory.Test) {
				if test.NeedsRegistry {
					opts := registry.Options{}
					registry := registry.New(helmCLI, c.GetKubeconfig())
					registry.Install(opts)
				}

				if test.NeedsSnapshots {
					GinkgoWriter.Println("Installing Minio")

					minio := minio.New(minio.Options{})
					minio.Install(helmCLI, c.GetKubeconfig())

					GinkgoWriter.Println("Installing Velero")

					veleroCLI.Install(w.GetDir(), c.GetKubeconfig(), minio)
				}

				if test.NeedsMonitoring {
					GinkgoWriter.Println("Installing Prometheus")

					prometheus := prometheus.New(prometheus.Options{})
					prometheus.Install(helmCLI, c.GetKubeconfig())
				}

				var adminConsolePort string
				if test.IsHelmManaged {
					GinkgoWriter.Println("Installing KOTS Helm chart")
					session, err := helmCLI.Install(c.GetKubeconfig(), "-n", test.Namespace, "admin-console", kotsHelmChartURL, "--set", fmt.Sprintf("password=%s", inventory.HelmPassword), "--version", kotsHelmChartVersion, "--create-namespace", "--wait")
					Expect(err).WithOffset(1).Should(Succeed(), "helm install")
					Eventually(session).WithOffset(1).WithTimeout(time.Minute).Should(gexec.Exit(0), "helm install failed with non-zero exit code")

					adminConsolePort = kotsInstaller.AdminConsolePortForward(c.GetKubeconfig(), test, kotsadmForwardPort)
				} else {
					GinkgoWriter.Println("Installing KOTS")
					adminConsolePort = kotsInstaller.Install(c.GetKubeconfig(), test, kotsadmForwardPort)
				}

				var testimParams inventory.TestimParams
				if test.Setup != nil {
					testimParams = test.Setup(kubectlCLI)
				}

				GinkgoWriter.Println("Running E2E tests")
				testimRun = testimClient.NewRun(c.GetKubeconfig(), test, testim.RunOptions{
					TunnelPort: adminConsolePort,
					BaseUrl:    testimBaseUrl,
					Params:     testimParams,
				})
				testimRun.ShouldSucceed()
			},
			func(test inventory.Test) string {
				return test.Name
			},
			Entry(nil, inventory.NewRegressionTest()),
			Entry(nil, inventory.NewSmokeTest()),
			Entry(nil, inventory.NewAirgapSmokeTest()),
			Entry(nil, inventory.NewConfigValidation()),
			Entry(nil, inventory.NewBackupAndRestore()),
			Entry(nil, inventory.NewNoRequiredConfig()),
			Entry(nil, inventory.NewVersionHistoryPagination()),
			Entry(nil, inventory.NewChangeLicense()),
			Entry(nil, inventory.NewHelmManagedMode()),
			Entry(nil, inventory.NewMinKotsVersion()),
			Entry(nil, inventory.NewTargetKotsVersion()),
			Entry(nil, inventory.NewRangeKotsVersion()),
			Entry(nil, inventory.NewMultiAppBackupAndRestoreTest()),
			Entry(nil, inventory.MultiAppTest()),
		)

	})

})
