package e2e

import (
	"context"
	"flag"
	"testing"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/ginkgo/v2"
	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/cluster"
	"github.com/replicatedhq/kots/e2e/helm"
	"github.com/replicatedhq/kots/e2e/inventory"
	"github.com/replicatedhq/kots/e2e/kots"
	"github.com/replicatedhq/kots/e2e/kubectl"
	"github.com/replicatedhq/kots/e2e/minio"
	"github.com/replicatedhq/kots/e2e/playwright"
	"github.com/replicatedhq/kots/e2e/prometheus"
	"github.com/replicatedhq/kots/e2e/registry"
	"github.com/replicatedhq/kots/e2e/util"
	"github.com/replicatedhq/kots/e2e/velero"
	"github.com/replicatedhq/kots/e2e/workspace"
)

var playwrightClient *playwright.Client
var helmCLI *helm.CLI
var veleroCLI *velero.CLI
var kotsInstaller *kots.Installer

var (
	skipTeardown          bool
	existingKubeconfig    string
	kotsadmImageRegistry  string
	kotsadmImageNamespace string
	kotsadmImageTag       string
	airgap                bool
	isOpenShift           bool
	isEKS                 bool
	kotsadmPort           string
	kotsHelmChartURL      string
	kotsHelmChartVersion  string
	kotsDockerhubUsername string
	kotsDockerhubPassword string
	awsAccessKeyID        string
	awsSecretAccessKey    string
	gitTag                string
)

func init() {
	flag.StringVar(&existingKubeconfig, "existing-kubeconfig", "", "use kubeconfig from existing cluster, do not create clusters (only for use with targeted testing)")
	flag.BoolVar(&skipTeardown, "skip-teardown", false, "do not tear down clusters")
	flag.StringVar(&kotsadmImageRegistry, "kotsadm-image-registry", "", "override the kotsadm images registry")
	flag.StringVar(&kotsadmImageNamespace, "kotsadm-image-namespace", "", "override the kotsadm images registry namespace")
	flag.StringVar(&kotsadmImageTag, "kotsadm-image-tag", "alpha", "override the kotsadm images tag")
	flag.BoolVar(&airgap, "airgap", false, "run install in airgapped mode")
	flag.BoolVar(&isOpenShift, "is-openshift", false, "the cluster is an openshift cluster")
	flag.BoolVar(&isEKS, "is-eks", false, "the cluster is an eks cluster")
	flag.StringVar(&kotsadmPort, "kotsadm-port", "", "sets the port that the admin console will be exposed on instead of generating a random one")
	flag.StringVar(&kotsHelmChartURL, "kots-helm-chart-url", "", "kots helm chart url")
	flag.StringVar(&kotsHelmChartVersion, "kots-helm-chart-version", "", "kots helm chart version")
	flag.StringVar(&kotsDockerhubUsername, "kots-dockerhub-username", "", "kots dockerhub username")
	flag.StringVar(&kotsDockerhubPassword, "kots-dockerhub-password", "", "kots dockerhub password")
	flag.StringVar(&awsAccessKeyID, "aws-access-key-id", "", "aws access key id")
	flag.StringVar(&awsSecretAccessKey, "aws-secret-access-key", "", "aws secret access key")
	flag.StringVar(&gitTag, "git-tag", "", "git tag")
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	Expect(util.CommandExists("kubectl")).To(BeTrue(), "kubectl required")
	Expect(util.CommandExists("helm")).To(BeTrue(), "helm required")
	Expect(util.CommandExists("velero")).To(BeTrue(), "velero required")
	Expect(util.CommandExists("kots")).To(BeTrue(), "kots required")

	w := workspace.New()
	DeferCleanup(w.Teardown)

	playwrightClient = playwright.NewClient(awsAccessKeyID, awsSecretAccessKey)

	helmCLI = helm.NewCLI(w.GetDir())

	veleroCLI = velero.NewCLI(w.GetDir(), isOpenShift)

	kotsInstaller = kots.NewInstaller(kotsadmImageRegistry, kotsadmImageNamespace, kotsadmImageTag, airgap, kotsDockerhubUsername, kotsDockerhubPassword, isEKS)
})

var _ = ReportBeforeSuite(func(report Report) {
	count := report.PreRunStats.SpecsThatWillRun
	if count == 0 {
		Fail("Did not match any tests")
	}
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

				if kotsadmPort == "" {
					var err error
					kotsadmPort, err = util.GetFreePort()
					Expect(err).WithOffset(1).Should(Succeed(), "get free port")
				}

				if !test.SkipKOTSInstall {
					GinkgoWriter.Println("Installing KOTS")
					kotsInstaller.Install(c.GetKubeconfig(), test, kotsadmPort)
				}

				GinkgoWriter.Println("Running E2E tests")

				if test.Setup != nil {
					test.Setup(kubectlCLI)
				}

				playwrightRun := playwrightClient.NewRun(c.GetKubeconfig(), test, playwright.RunOptions{
					Port: kotsadmPort,
				})
				playwrightRun.ShouldSucceed()
			},
			func(test inventory.Test) string {
				return test.ID
			},
			Entry(nil, inventory.NewRegressionTest(gitTag)),
			Entry(nil, inventory.NewSmokeTestOnline()),
			Entry(nil, inventory.NewSmokeTestAirgap()),
			Entry(nil, inventory.NewConfigValidation()),
			Entry(nil, inventory.NewBackupAndRestore()),
			Entry(nil, inventory.NewNoRequiredConfig()),
			Entry(nil, inventory.NewVersionHistoryPagination()),
			Entry(nil, inventory.NewChangeLicense()),
			Entry(nil, inventory.NewMinKotsVersionOnline()),
			Entry(nil, inventory.NewMinKotsVersionAirgap()),
			Entry(nil, inventory.NewTargetKotsVersionOnline()),
			Entry(nil, inventory.NewTargetKotsVersionAirgap()),
			Entry(nil, inventory.NewRangeKotsVersionOnline()),
			Entry(nil, inventory.NewRangeKotsVersionAirgap()),
			Entry(nil, inventory.NewMultiAppBackupAndRestoreTest()),
			Entry(nil, inventory.MultiAppTest()),
			Entry(nil, inventory.NewSupportBundle()),
			Entry(nil, inventory.NewGitOps()),
			Entry(nil, inventory.NewChangeChannel()),
		)
	})
})
