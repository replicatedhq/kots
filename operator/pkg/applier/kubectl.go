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
	stdoutCh := make(chan []byte)
	stderrCh := make(chan []byte)
	stopCh := make(chan bool)

	cmd.Dir = "/tmp"

	stdout := [][]byte{}
	stderr := [][]byte{}

	defer func() {
		stopCh <- true
	}()

	go func() {
		for {
			select {
			case o := <-stdoutCh:
				stdout = append(stdout, o)
			case e := <-stderrCh:
				stderr = append(stderr, e)
			case <-stopCh:
				return
			}
		}
	}()

	if err := Run(cmd, &stdoutCh, &stderrCh); err != nil {
		log.Printf("error running kubectl support-bundle: \n stderr %s\n stdout %s", bytes.Join(stderr, []byte("\n")), bytes.Join(stdout, []byte("\n")))
		return errors.Wrap(err, "failed to run kubectl support-bundle")
	}

	return nil
}

func (c *Kubectl) Preflight(preflightURI string) error {
	log.Printf("running kubectl preflight %s", preflightURI)
	args := []string{
		preflightURI,
	}

	cmd := c.preflightCommand(args...)
	cmd.Env = os.Environ()
	stdoutCh := make(chan []byte)
	stderrCh := make(chan []byte)
	stopCh := make(chan bool)

	stdout := [][]byte{}
	stderr := [][]byte{}

	defer func() {
		stopCh <- true
	}()

	go func() {
		for {
			select {
			case o := <-stdoutCh:
				stdout = append(stdout, o)
			case e := <-stderrCh:
				stderr = append(stderr, e)
			case <-stopCh:
				return
			}
		}
	}()

	if err := Run(cmd, &stdoutCh, &stderrCh); err != nil {
		log.Printf("error running kubectl preflight: \n stderr %s\n stdout %s", bytes.Join(stderr, []byte("\n")), bytes.Join(stdout, []byte("\n")))
		return errors.Wrap(err, "failed to run kubectl preflight")
	}

	return nil
}

func (c *Kubectl) Apply(targetNamespace string, yamlDoc []byte, dryRun bool) ([]byte, []byte, error) {
	args := []string{
		"apply",
	}

	if dryRun {
		args = append(args, "--dry-run")
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

	stdoutCh := make(chan []byte)
	stderrCh := make(chan []byte)
	stopCh := make(chan bool)

	stdout := [][]byte{}
	stderr := [][]byte{}

	defer func() {
		stopCh <- true
	}()

	go func() {
		for {
			select {
			case o := <-stdoutCh:
				stdout = append(stdout, o)
			case e := <-stderrCh:
				stderr = append(stderr, e)
			case <-stopCh:
				return
			}
		}
	}()

	if err := Run(cmd, &stdoutCh, &stderrCh); err != nil {
		return bytes.Join(stdout, []byte("\n")), bytes.Join(stderr, []byte("\n")), errors.Wrap(err, "failed to run kubectl apply")
	}

	return bytes.Join(stdout, []byte("\n")), bytes.Join(stderr, []byte("\n")), nil
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
