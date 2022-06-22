package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/kubectl"
)

func regressionCreateRegistryCredsSecret(kubectlCLI *kubectl.CLI) {
	cmd := kubectlCLI.Command(
		context.Background(),
		"create",
		"secret",
		"docker-registry",
		"registry-creds",
		"--docker-server=registry.default.svc.cluster.local:5000",
		"--docker-username=fake",
		"--docker-password=fake",
		"--docker-email=fake@fake.com",
	)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).WithOffset(1).Should(Succeed(), "Create registry-creds secret failed")
	Eventually(session).WithOffset(1).WithTimeout(30*time.Minute).Should(gexec.Exit(0), "Create registry-creds secret failed with non-zero exit code")
}
