package applier

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	rest "k8s.io/client-go/rest"
)

type Kubectl struct {
	kubectl       string
	preflight     string
	supportBundle string
	config        *rest.Config
}

func NewKubectl(kubectl string, preflight string, supportBundle string, config *rest.Config) *Kubectl {
	return &Kubectl{
		kubectl:       kubectl,
		preflight:     preflight,
		supportBundle: supportBundle,
		config:        config,
	}
}

// Thanks weaveworks/flux
func (c *Kubectl) connectArgs() []string {
	var args []string
	if c.config.Host != "" {
		args = append(args, fmt.Sprintf("--server=%s", c.config.Host))
	}
	if c.config.Username != "" {
		args = append(args, fmt.Sprintf("--username=%s", c.config.Username))
	}
	if c.config.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", c.config.Password))
	}
	if c.config.TLSClientConfig.CertFile != "" {
		args = append(args, fmt.Sprintf("--client-certificate=%s", c.config.TLSClientConfig.CertFile))
	}
	if c.config.TLSClientConfig.CAFile != "" {
		args = append(args, fmt.Sprintf("--certificate-authority=%s", c.config.TLSClientConfig.CAFile))
	}
	if c.config.TLSClientConfig.KeyFile != "" {
		args = append(args, fmt.Sprintf("--client-key=%s", c.config.TLSClientConfig.KeyFile))
	}
	if c.config.BearerToken != "" {
		args = append(args, fmt.Sprintf("--token=%s", c.config.BearerToken))
	}
	return args
}

func (c *Kubectl) SupportBundle(collectorURI string) error {
	log.Printf("running kubectl supportBundle %s", collectorURI)
	args := []string{
		collectorURI,
	}

	cmd := c.supportBundleCommand(args...)
	cmd.Env = os.Environ()
	cmd.Dir = "/tmp"

	stdout, stderr, err := Run(cmd)
	if err != nil {
		log.Printf("error running kubectl support-bundle: \n stderr %s\n stdout %s", stderr, stdout)
		return errors.Wrap(err, "failed to run kubectl support-bundle")
	}

	return nil
}

func (c *Kubectl) Preflight(preflightURI string, ignorePermissions bool) error {
	log.Printf("running kubectl preflight %s", preflightURI)

	args := []string{}
	if ignorePermissions {
		args = append(args, "--collect-without-permissions=true")
	}
	args = append(args, preflightURI)

	cmd := c.preflightCommand(args...)
	cmd.Env = os.Environ()

	stdout, stderr, err := Run(cmd)
	if err != nil {
		log.Printf("error running kubectl preflight: \n stderr %s\n stdout %s", stderr, stdout)
		return errors.Wrap(err, "failed to run kubectl preflight")
	}

	return nil
}

func (c *Kubectl) Remove(targetNamespace string, yamlDoc []byte, wait bool) ([]byte, []byte, error) {
	args := []string{
		"delete",
	}

	if wait {
		args = append(args, "--wait")
	}

	if targetNamespace != "" {
		args = append(args, []string{
			"-n",
			targetNamespace,
		}...)
	}

	args = append(args, []string{
		"-f",
		"-",
	}...)

	cmd := c.kubectlCommand(args...)
	cmd.Stdin = bytes.NewReader(yamlDoc)

	stdout, stderr, err := Run(cmd)
	return stdout, stderr, errors.Wrap(err, "failed to run kubectl delete")
}

func (c *Kubectl) Apply(targetNamespace string, yamlDoc []byte, dryRun bool, wait bool) ([]byte, []byte, error) {
	args := []string{
		"apply",
	}

	if dryRun {
		args = append(args, "--dry-run")
	}
	if wait {
		args = append(args, "--wait")
	}

	if targetNamespace != "" {
		args = append(args, []string{
			"-n",
			targetNamespace,
		}...)
	}

	args = append(args, []string{
		"-f",
		"-",
	}...)

	cmd := c.kubectlCommand(args...)
	cmd.Stdin = bytes.NewReader(yamlDoc)

	stdout, stderr, err := Run(cmd)
	return stdout, stderr, errors.Wrap(err, "failed to run kubectl apply")
}

func (c *Kubectl) kubectlCommand(args ...string) *exec.Cmd {
	return exec.Command(c.kubectl, append(args, c.connectArgs()...)...)
}

func (c *Kubectl) supportBundleCommand(args ...string) *exec.Cmd {
	if c.supportBundle != "" {
		allArgs := append(args, c.connectArgs()...)
		return exec.Command(c.supportBundle, allArgs...)
	}

	allArgs := append([]string{"support-bundle"}, args...)
	allArgs = append(allArgs, c.connectArgs()...)
	return exec.Command(c.kubectl, allArgs...)
}

func (c *Kubectl) preflightCommand(args ...string) *exec.Cmd {
	if c.preflight != "" {
		allArgs := append([]string{"--interactive=false"}, args...)
		allArgs = append(allArgs, c.connectArgs()...)
		return exec.Command(c.preflight, allArgs...)
	}

	allArgs := append([]string{"preflight", "--interactive=false"}, args...)
	allArgs = append(allArgs, c.connectArgs()...)
	return exec.Command(c.kubectl, allArgs...)
}
