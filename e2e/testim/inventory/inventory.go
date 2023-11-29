package inventory

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"
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
		Label:           "type=existing cluster, env=online, phase=new install, rbac=minimal rbac",
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
		Name:           "Smoke Test",
		Suite:          "smoke-test",
		Namespace:      "smoke-test",
		UpstreamURI:    "qakotstestim/github-actions-qa",
		NeedsSnapshots: true,
	}
}

func NewAirgapSmokeTest() Test {
	return Test{
		Name:        "airgap-smoke-test",
		Suite:       "airgap-smoke-test",
		Namespace:   "airgap-smoke-test",
		UpstreamURI: "airgap-smoke-test/automated",
	}
}

func NewConfigValidation() Test {
	return Test{
		Name:        "Config Validation",
		Suite:       "config-validation",
		Namespace:   "config-validation",
		UpstreamURI: "config-validation-panda/automated",
	}
}

func NewBackupAndRestore() Test {
	return Test{
		Name:           "Backup and Restore",
		Suite:          "backup-and-restore",
		Namespace:      "backup-and-restore",
		UpstreamURI:    "backup-and-restore/automated",
		NeedsSnapshots: true,
	}
}

func NewNoRequiredConfig() Test {
	return Test{
		Name:        "No Required Config",
		Suite:       "no-required-config",
		Namespace:   "no-required-config",
		UpstreamURI: "no-required-config/automated",
		Setup:       SetupNoRequiredConfig,
	}
}

func NewVersionHistoryPagination() Test {
	return Test{
		Name:        "Version History Pagination",
		Suite:       "version-history-pagination",
		Namespace:   "version-history-pagination",
		UpstreamURI: "version-history-pagination/automated",
	}
}

func NewChangeLicense() Test {
	return Test{
		Name:        "Change License",
		Suite:       "change-license",
		Namespace:   "change-license",
		UpstreamURI: "change-license/automated",
	}
}

func NewHelmManagedMode() Test {
	return Test{
		Name:          "Helm Managed",
		Suite:         "helm-managed",
		Namespace:     "helm-managed",
		UpstreamURI:   "helm-managed/automated",
		IsHelmManaged: true,
		Setup:         SetupHelmManagedMode,
	}
}

func NewMultiAppBackupAndRestoreTest() Test {
	return Test{
		Name:           "multi-app-backup-and-restore",
		Suite:          "multi-app-backup-and-restore",
		Namespace:      "multi-app-backup-and-restore",
		UpstreamURI:    "multi-app-backup-and-restore/automated",
		NeedsSnapshots: true,
	}
}

func MultiAppTest() Test {
	return Test{
		Name:        "multi-app-install",
		Suite:       "multi-app-install",
		Namespace:   "multi-app-install",
		UpstreamURI: "multi-app-install/automated",
	}
}

func NewMinKotsVersion() Test {
	return Test{
		Name:                   "Min KOTS Version",
		Suite:                  "min-kots-version",
		Namespace:              "min-kots-version",
		UpstreamURI:            "min-kots-version/automated",
		SkipCompatibilityCheck: true,
	}
}

func NewTargetKotsVersion() Test {
	return Test{
		Name:                   "Target KOTS Version",
		Suite:                  "target-kots-version",
		Namespace:              "target-kots-version",
		UpstreamURI:            "target-kots-version/automated",
		SkipCompatibilityCheck: true,
	}
}

func NewRangeKotsVersion() Test {
	return Test{
		Name:                   "Range KOTS Version",
		Suite:                  "range-kots-version",
		Namespace:              "range-kots-version",
		UpstreamURI:            "range-kots-version/automated",
		SkipCompatibilityCheck: true,
	}
}

func NewSupportBundle() Test {
	return Test{
		Name:        "Support Bundle",
		Suite:       "support-bundle",
		Namespace:   "support-bundle",
		UpstreamURI: "support-bundle-halibut/automated",
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

func SetupNoRequiredConfig(kubectlCLI *kubectl.CLI) TestimParams {
	cmd := kubectlCLI.Command(
		context.Background(),
		"--namespace=no-required-config",
		"get",
		"secret",
		"kotsadm-authstring",
		`--template='{{ index .data "kotsadm-authstring" }}'`,
	)
	buf := bytes.NewBuffer(nil)
	session, err := gexec.Start(cmd, buf, GinkgoWriter)
	Expect(err).WithOffset(1).Should(Succeed(), "Get kotsadm-authstring secret failed")
	Eventually(session).WithOffset(1).WithTimeout(30*time.Minute).Should(gexec.Exit(0), "Get kotsadm-authstring secret failed with non-zero exit code")

	kotsadmAPIToken, err := base64.StdEncoding.DecodeString(strings.Trim(buf.String(), `"' `))
	Expect(err).WithOffset(1).Should(Succeed(), "Decode kotsadm-authstring secret failed")

	err = ioutil.WriteFile(".env", []byte(fmt.Sprintf("KOTSADM_API_TOKEN=%s", string(kotsadmAPIToken))), 0600)
	Expect(err).WithOffset(1).Should(Succeed(), "Create .env file failed")
	return nil
}

func SetupHelmManagedMode(kubectlCLI *kubectl.CLI) TestimParams {
	return TestimParams{
		"kotsadmPassword":  HelmPassword,
		"kotsadmNamespace": "helm-managed",
	}
}
