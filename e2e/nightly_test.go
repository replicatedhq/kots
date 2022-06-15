package e2e

import (
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/kubectl"
)

func nightlyCreateRegistryCredsSecret(kubectlCLI *kubectl.CLI) {
	session, err := kubectlCLI.RunCommand(
		"create",
		"secret",
		"docker-registry",
		"registry-creds",
		"--docker-server=registry.default.svc.cluster.local:5000",
		"--docker-username=fake",
		"--docker-password=fake",
		"--docker-email=fake@fake.com",
	)
	Expect(err).WithOffset(1).Should(Succeed(), "Create registry-creds secret failed")
	Eventually(session).WithOffset(1).WithTimeout(30*time.Minute).Should(gexec.Exit(0), "Create registry-creds secret failed with non-zero exit code")
}
