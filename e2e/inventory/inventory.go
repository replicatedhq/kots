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
		ID:              "@regression",
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
		ID:             "@smoke-test",
		Namespace:      "smoke-test",
		AppSlug:        "qakotstestim",
		UpstreamURI:    "qakotstestim/github-actions-qa",
		NeedsSnapshots: true,
	}
}

func NewAirgapSmokeTest() Test {
	return Test{
		ID:          "@airgap-smoke-test",
		TestimSuite: "airgap-smoke-test",
		Namespace:   "airgap-smoke-test",
		UpstreamURI: "airgap-smoke-test/automated",
	}
}

func NewConfigValidation() Test {
	return Test{
		ID:          "@config-validation",
		Namespace:   "config-validation",
		AppSlug:     "config-validation-panda",
		UpstreamURI: "config-validation-panda/automated",
	}
}

func NewBackupAndRestore() Test {
	return Test{
		ID:             "@backup-and-restore",
		Namespace:      "backup-and-restore",
		AppSlug:        "backup-and-restore",
		UpstreamURI:    "backup-and-restore/automated",
		NeedsSnapshots: true,
	}
}

func NewNoRequiredConfig() Test {
	return Test{
		ID:          "@no-required-config",
		Namespace:   "no-required-config",
		AppSlug:     "no-required-config",
		UpstreamURI: "no-required-config/automated",
	}
}

func NewVersionHistoryPagination() Test {
	return Test{
		ID:          "@version-history-pagination",
		TestimSuite: "version-history-pagination",
		Namespace:   "version-history-pagination",
		UpstreamURI: "version-history-pagination/automated",
	}
}

func NewChangeLicense() Test {
	return Test{
		ID:          "@change-license",
		Namespace:   "change-license",
		AppSlug:     "change-license",
		UpstreamURI: "change-license/automated",
	}
}

func NewMultiAppBackupAndRestoreTest() Test {
	return Test{
		ID:             "@multi-app-backup-and-restore",
		Namespace:      "multi-app-backup-and-restore",
		AppSlug:        "multi-app-backup-and-restore",
		UpstreamURI:    "multi-app-backup-and-restore/automated",
		NeedsSnapshots: true,
	}
}

func MultiAppTest() Test {
	return Test{
		ID:          "@multi-app-install",
		Namespace:   "multi-app-install",
		AppSlug:     "mutli-app-install",
		UpstreamURI: "mutli-app-install/automated",
	}
}

func NewMinKotsVersionOnline() Test {
	return Test{
		ID:                     "@min-kots-version-online",
		Namespace:              "min-kots-version",
		AppSlug:                "min-kots-version",
		UpstreamURI:            "min-kots-version/automated",
		SkipCompatibilityCheck: true,
	}
}

func NewMinKotsVersionAirgap() Test {
	return Test{
		ID:                     "@min-kots-version-airgap",
		Namespace:              "min-kots-version",
		AppSlug:                "min-kots-version",
		UpstreamURI:            "min-kots-version/automated",
		SkipCompatibilityCheck: true,
	}
}

func NewTargetKotsVersion() Test {
	return Test{
		ID:                     "@target-kots-version",
		TestimSuite:            "target-kots-version",
		Namespace:              "target-kots-version",
		UpstreamURI:            "target-kots-version/automated",
		SkipCompatibilityCheck: true,
	}
}

func NewRangeKotsVersion() Test {
	return Test{
		ID:                     "@range-kots-version",
		TestimSuite:            "range-kots-version",
		Namespace:              "range-kots-version",
		UpstreamURI:            "range-kots-version/automated",
		SkipCompatibilityCheck: true,
	}
}

func NewSupportBundle() Test {
	return Test{
		ID:          "@support-bundle",
		TestimSuite: "support-bundle",
		Namespace:   "support-bundle",
		UpstreamURI: "support-bundle-halibut/automated",
	}
}

func NewGitOps() Test {
	return Test{
		ID:          "@gitops",
		TestimSuite: "gitops",
		Namespace:   "gitops",
		UpstreamURI: "gitops-bobcat/automated",
	}
}

func NewChangeChannel() Test {
	return Test{
		ID:          "@change-channel",
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
