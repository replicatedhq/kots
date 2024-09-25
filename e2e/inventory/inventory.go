package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/kubectl"
	"github.com/replicatedhq/kots/e2e/registry"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/ginkgo/v2"
	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
)

const (
	HelmPassword = "password"
)

func NewRegressionTest() Test {
	return Test{
		Name:            "Regression",
		TestimLabel:     "type=existing cluster, env=online, phase=new install, rbac=minimal rbac",
		Namespace:       "qakotsregression",
		UpstreamURI:     "qakotsregression/type-existing-cluster-env-on-2",
		Browser:         "firefox",
		UseMinimalRBAC:  true,
		NeedsMonitoring: true,
		NeedsRegistry:   true,
		Setup:           SetupRegressionTest,
	}
}

func NewSmokeTest() Test {
	return Test{
		ID:             "smoke-test",
		Name:           "Smoke Test",
		Namespace:      "smoke-test",
		AppSlug:        "qakotstestim",
		UpstreamURI:    "qakotstestim/github-actions-qa",
		NeedsSnapshots: true,
	}
}

func NewAirgapSmokeTest() Test {
	return Test{
		Name:        "airgap-smoke-test",
		TestimSuite: "airgap-smoke-test",
		Namespace:   "airgap-smoke-test",
		UpstreamURI: "airgap-smoke-test/automated",
	}
}

func NewConfigValidation() Test {
	return Test{
		ID:          "config-validation",
		Name:        "Config Validation",
		Namespace:   "config-validation",
		AppSlug:     "config-validation-panda",
		UpstreamURI: "config-validation-panda/automated",
	}
}

func NewBackupAndRestore() Test {
	return Test{
		ID:             "backup-and-restore",
		Name:           "Backup and Restore",
		Namespace:      "backup-and-restore",
		AppSlug:        "backup-and-restore",
		UpstreamURI:    "backup-and-restore/automated",
		NeedsSnapshots: true,
	}
}

func NewNoRequiredConfig() Test {
	return Test{
		ID:          "no-required-config",
		Name:        "No Required Config",
		Namespace:   "no-required-config",
		AppSlug:     "no-required-config",
		UpstreamURI: "no-required-config/automated",
	}
}

func NewVersionHistoryPagination() Test {
	return Test{
		Name:        "Version History Pagination",
		TestimSuite: "version-history-pagination",
		Namespace:   "version-history-pagination",
		UpstreamURI: "version-history-pagination/automated",
	}
}

func NewChangeLicense() Test {
	return Test{
		Name:        "Change License",
		TestimSuite: "change-license",
		Namespace:   "change-license",
		UpstreamURI: "change-license/automated",
	}
}

func NewMultiAppBackupAndRestoreTest() Test {
	return Test{
		Name:           "multi-app-backup-and-restore",
		TestimSuite:    "multi-app-backup-and-restore",
		Namespace:      "multi-app-backup-and-restore",
		UpstreamURI:    "multi-app-backup-and-restore/automated",
		NeedsSnapshots: true,
	}
}

func MultiAppTest() Test {
	return Test{
		ID:          "multi-app-install",
		Name:        "Multi App Install",
		Namespace:   "multi-app-install",
		AppSlug:     "mutli-app-install",
		UpstreamURI: "mutli-app-install/automated",
	}
}

func NewMinKotsVersion() Test {
	return Test{
		Name:                   "Min KOTS Version",
		TestimSuite:            "min-kots-version",
		Namespace:              "min-kots-version",
		UpstreamURI:            "min-kots-version/automated",
		SkipCompatibilityCheck: true,
	}
}

func NewTargetKotsVersion() Test {
	return Test{
		Name:                   "Target KOTS Version",
		TestimSuite:            "target-kots-version",
		Namespace:              "target-kots-version",
		UpstreamURI:            "target-kots-version/automated",
		SkipCompatibilityCheck: true,
	}
}

func NewRangeKotsVersion() Test {
	return Test{
		Name:                   "Range KOTS Version",
		TestimSuite:            "range-kots-version",
		Namespace:              "range-kots-version",
		UpstreamURI:            "range-kots-version/automated",
		SkipCompatibilityCheck: true,
	}
}

func NewSupportBundle() Test {
	return Test{
		Name:        "Support Bundle",
		TestimSuite: "support-bundle",
		Namespace:   "support-bundle",
		UpstreamURI: "support-bundle-halibut/automated",
	}
}

func NewGitOps() Test {
	return Test{
		Name:        "GitOps",
		TestimSuite: "gitops",
		Namespace:   "gitops",
		UpstreamURI: "gitops-bobcat/automated",
	}
}

func NewChangeChannel() Test {
	return Test{
		ID:          "change-channel",
		Name:        "Change Channel",
		Namespace:   "change-channel",
		AppSlug:     "change-channel",
		UpstreamURI: "change-channel/automated",
	}
}

func SetupRegressionTest(kubectlCLI *kubectl.CLI) TestimParams {
	cmd := kubectlCLI.Command(
		context.Background(),
		"create",
		"secret",
		"docker-registry",
		"registry-creds",
		fmt.Sprintf("--docker-server=registry.%s.svc.cluster.local:5000", registry.DefaultNamespace),
		"--docker-username=fake",
		"--docker-password=fake",
		"--docker-email=fake@fake.com",
	)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).WithOffset(1).Should(Succeed(), "Create registry-creds secret failed")
	Eventually(session).WithOffset(1).WithTimeout(30*time.Minute).Should(gexec.Exit(0), "Create registry-creds secret failed with non-zero exit code")
	return nil
}
