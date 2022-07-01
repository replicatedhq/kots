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

func NewRegressionECONIMRTest() Test {
	return Test{
		Name:            "Regression: Existing Cluster, Online, New Install, Minimal RBAC",
		Label:           "type=existing cluster, env=online, phase=new install, rbac=minimal rbac",
		Namespace:       "qakotsregression",
		UpstreamURI:     "qakotsregression/type-existing-cluster-env-on-2",
		UseMinimalRBAC:  true,
		NeedsMonitoring: true,
		NeedsRegistry:   true,
		Setup:           SetupRegressionTest,
	}
}

func NewRegressionECONICATest() Test {
	return Test{
		Name:            "Regression: Existing Cluster, Online, New Install, Cluster Admin",
		Label:           "type=existing cluster, env=online, phase=new install, rbac=cluster admin",
		Namespace:       "qakotsregression",
		UpstreamURI:     "qakotsregression/type-existing-cluster-env-onli",
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

func NewStrictPreflightChecks() Test {
	return Test{
		Name:        "Strict Preflight Checks",
		Suite:       "strict-preflight-checks",
		Namespace:   "strict-preflight-checks",
		UpstreamURI: "strict-preflight-checks/automated",
	}
}

func NewMinimalRBACTest() Test {
	return Test{
		Name:        "Minimal RBAC",
		Suite:       "minimal-rbac",
		Namespace:   "minimal-rbac",
		UpstreamURI: "minimal-rbac/automated",
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

func SetupRegressionTest(kubectlCLI *kubectl.CLI) {
	cmd := kubectlCLI.Command(
		context.Background(),
		fmt.Sprintf("--namespace=%s", registry.DefaultNamespace),
		"get",
		"svc",
		registry.DefaultReleaseName,
		"--template={{ .spec.clusterIP }}",
	)
	out, err := cmd.Output()
	Expect(err).WithOffset(1).Should(Succeed(), "Get registry cluster ip failed")
	clusterIP := strings.TrimSpace(string(out))

	cmd = kubectlCLI.Command(
		context.Background(),
		"create",
		"secret",
		"docker-registry",
		"registry-creds",
		fmt.Sprintf("--docker-server=%s:5000", clusterIP),
		"--docker-username=fake",
		"--docker-password=fake",
		"--docker-email=fake@fake.com",
	)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).WithOffset(1).Should(Succeed(), "Create registry-creds secret failed")
	Eventually(session).WithOffset(1).WithTimeout(30*time.Minute).Should(gexec.Exit(0), "Create registry-creds secret failed with non-zero exit code")
}

func SetupNoRequiredConfig(kubectlCLI *kubectl.CLI) {
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
}
