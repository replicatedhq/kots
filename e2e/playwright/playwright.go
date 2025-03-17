package playwright

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/inventory"
	"github.com/replicatedhq/kots/e2e/util"
)

type Client struct {
	imageRegistry     string
	imageNamespace    string
	imageTag          string
	dockerhubUsername string
	dockerhubPassword string
}

type Run struct {
	session *gexec.Session
}

type RunOptions struct {
	Port string
}

func NewClient(imageRegistry, imageNamespace, imageTag string, dockerhubUsername string, dockerhubPassword string) *Client {
	return &Client{
		imageRegistry:     imageRegistry,
		imageNamespace:    imageNamespace,
		imageTag:          imageTag,
		dockerhubUsername: dockerhubUsername,
		dockerhubPassword: dockerhubPassword,
	}
}

func (t *Client) HasTest(test inventory.Test) bool {
	if test.ID == "" {
		return false
	}
	_, err := os.Stat(fmt.Sprintf("/playwright/%s", test.Path()))
	return err == nil
}

func (t *Client) NewRun(kubeconfig string, test inventory.Test, runOptions RunOptions) *Run {
	args := []string{
		"playwright",
		"test",
		test.ID,
	}

	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = test.Namespace
	}

	cmd := exec.Command("npx", args...)
	cmd.Dir = "/playwright"
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%s", runOptions.Port))
	cmd.Env = append(cmd.Env, fmt.Sprintf("NAMESPACE=%s", namespace))
	cmd.Env = append(cmd.Env, fmt.Sprintf("APP_SLUG=%s", test.AppSlug))
	cmd.Env = append(cmd.Env, fmt.Sprintf("TEST_DIR=%s", test.Dir()))
	cmd.Env = append(cmd.Env, fmt.Sprintf("TEST_PATH=%s", test.Path()))
	cmd.Env = append(cmd.Env, fmt.Sprintf("KOTSADM_IMAGE_REGISTRY=%s", t.imageRegistry))
	cmd.Env = append(cmd.Env, fmt.Sprintf("KOTSADM_IMAGE_NAMESPACE=%s", t.imageNamespace))
	cmd.Env = append(cmd.Env, fmt.Sprintf("KOTSADM_IMAGE_TAG=%s", t.imageTag))
	cmd.Env = append(cmd.Env, fmt.Sprintf("KOTSADM_DOCKERHUB_USERNAME=%s", t.dockerhubUsername))
	cmd.Env = append(cmd.Env, fmt.Sprintf("KOTSADM_DOCKERHUB_PASSWORD=%s", t.dockerhubPassword))
	cmd.Env = append(cmd.Env, "NODE_OPTIONS=--max-old-space-size=4096")
	cmd.Env = append(cmd.Env, "DISABLE_SNAPSHOTS_VOLUME_ASSERTIONS=true")
	session, err := util.RunCommand(cmd)
	Expect(err).WithOffset(1).Should(Succeed(), "Run playwright test failed")
	return &Run{session}
}

func (r *Run) ShouldSucceed() {
	Eventually(r.session).WithOffset(1).WithTimeout(30*time.Minute).Should(gexec.Exit(), "Run playwright test timed out")
	Expect(r.session.ExitCode()).Should(Equal(0), "Run playwright test failed with non-zero exit code")
}
