package kots

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/e2e/inventory"
	"github.com/replicatedhq/kots/e2e/util"
)

var (
	InstallWaitDuration = 5 * time.Minute
)

type Installer struct {
	imageRegistry     string
	imageNamespace    string
	imageTag          string
	airgap            bool
	dockerhubUsername string
	dockerhubPassword string
	isEKS             bool
}

func NewInstaller(imageRegistry, imageNamespace, imageTag string, airgap bool, dockerhubUsername string, dockerhubPassword string, isEKS bool) *Installer {
	return &Installer{
		imageRegistry:     imageRegistry,
		imageNamespace:    imageNamespace,
		imageTag:          imageTag,
		airgap:            airgap,
		dockerhubUsername: dockerhubUsername,
		dockerhubPassword: dockerhubPassword,
		isEKS:             isEKS,
	}
}

func (i *Installer) Install(kubeconfig string, test inventory.Test, adminConsolePort string) {
	session, err := i.install(kubeconfig, test)
	Expect(err).WithOffset(1).Should(Succeed(), "Kots install failed")
	Eventually(session).WithOffset(1).WithTimeout(InstallWaitDuration).Should(gexec.Exit(0), "Kots install failed with non-zero exit code")

	if i.dockerhubUsername != "" && i.dockerhubPassword != "" {
		session, err = i.ensureSecret(kubeconfig, test)
		Expect(err).WithOffset(1).Should(Succeed(), "Kots docker ensure-secret failed")
		Eventually(session).WithOffset(1).WithTimeout(InstallWaitDuration).Should(gexec.Exit(0), "Kots docker ensure-secret failed with non-zero exit code")
	}

	i.AdminConsolePortForward(kubeconfig, test, adminConsolePort)
}

func (i *Installer) AdminConsolePortForward(kubeconfig string, test inventory.Test, adminConsolePort string) {
	var err error
	for x := 0; x < 3; x++ {
		err = i.portForward(kubeconfig, test.Namespace, adminConsolePort)
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	Expect(err).WithOffset(1).Should(Succeed(), "port forward")
}

func (i *Installer) ensureSecret(kubeconfig string, test inventory.Test) (*gexec.Session, error) {
	args := []string{
		"docker",
		"ensure-secret",
		fmt.Sprintf("--kubeconfig=%s", kubeconfig),
		fmt.Sprintf("--dockerhub-username=%s", i.dockerhubUsername),
		fmt.Sprintf("--dockerhub-password=%s", i.dockerhubPassword),
		fmt.Sprintf("--namespace=%s", test.Namespace),
	}

	return util.RunCommand(exec.Command("kots", args...))
}

func (i *Installer) install(kubeconfig string, test inventory.Test) (*gexec.Session, error) {
	args := []string{
		"install",
		test.UpstreamURI,
		fmt.Sprintf("--kubeconfig=%s", kubeconfig),
		"--no-port-forward",
		fmt.Sprintf("--namespace=%s", test.Namespace),
		"--shared-password=password",
		fmt.Sprintf("--kotsadm-registry=%s", i.imageRegistry),
		fmt.Sprintf("--kotsadm-namespace=%s", i.imageNamespace),
		fmt.Sprintf("--kotsadm-tag=%s", i.imageTag),
		fmt.Sprintf("--airgap=%t", i.airgap),
		fmt.Sprintf("--wait-duration=%s", InstallWaitDuration),
		fmt.Sprintf("--use-minimal-rbac=%t", test.UseMinimalRBAC),
		fmt.Sprintf("--skip-compatibility-check=%t", test.SkipCompatibilityCheck),
	}
	if i.isEKS {
		args = append(args, "--storage-class=gp2")
	}

	return util.RunCommand(exec.Command("kots", args...))
}

func (i *Installer) portForward(kubeconfig, namespace, adminConsolePort string) error {
	url := fmt.Sprintf("http://localhost:%s", adminConsolePort)

	timeout := time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		defer cancel()
		_, err := util.RunCommand(exec.Command(
			"kots",
			"admin-console",
			fmt.Sprintf("--kubeconfig=%s", kubeconfig),
			fmt.Sprintf("--namespace=%s", namespace),
			fmt.Sprintf("--port=%s", adminConsolePort),
			fmt.Sprintf("--wait-duration=%s", timeout),
		))
		Expect(err).WithOffset(1).Should(Succeed(), "async port forward")
	}()

	var err error
	for {
		select {
		case <-time.After(2 * time.Second):
			_, err = http.Get(fmt.Sprintf("%s/api/v1/ping", url))
			if err == nil {
				return nil
			}
		case <-ctx.Done():
			return errors.Wrap(err, "api ping timeout")
		}
	}
}
