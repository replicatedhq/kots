package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/registry"
	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/kubectl"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/ginkgo/v2"
	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
)

func NewChangeLicense() Test {
	return Test{
		Name:        "Change License",
		Suite:       "change-license",
		Namespace:   "change-license",
		UpstreamURI: "change-license/automated",
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

func NewRegressionTest() Test {
	return Test{
		Name:            "Regression",
		Label:           "type=existing cluster, env=online, phase=new install, rbac=minimal rbac",
		Namespace:       "qakotsregression",
		UpstreamURI:     "qakotsregression/type-existing-cluster-env-on-2",
		UseMinimalRBAC:  true,
		NeedsMonitoring: true,
		NeedsRegistry:   true,
		Setup:           SetupRegressionTest,
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
		Name:        "Minimal RBAC App",
		Suite:       "minimal-rbac",
		Namespace:   "minimal-rbac",
		UpstreamURI: "minimal-rbac/automated",
	}
}

func NewMinimalRBACOverrideTest() Test {
	return Test{
		Name:           "Minimal RBAC Override",
		Suite:          "minimal-rbac",
		Namespace:      "minimal-rbac",
		UpstreamURI:    "minimal-rbac/automated",
		UseMinimalRBAC: true,
	}
}

func SetupRegressionTest(kubectlCLI *kubectl.CLI) {
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
}
