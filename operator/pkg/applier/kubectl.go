package applier

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	rest "k8s.io/client-go/rest"
)

type Kubectl struct {
	exe    string
	config *rest.Config
}

func NewKubectl(exe string, config *rest.Config) *Kubectl {
	return &Kubectl{
		exe:    exe,
		config: config,
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

func (c *Kubectl) Remove(namespace string, yamlDoc []byte) error {
	args := []string{
		"delete",
		"-n",
		namespace,
		"-f",
		"-",
	}
	cmd := c.kubectlCommand(args...)
	cmd.Stdin = bytes.NewReader(yamlDoc)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	cmd.Stdout = &bytes.Buffer{}

	if err := cmd.Run(); err != nil {
		fmt.Printf("error running kubectl delete: %q\n", err)
		return errors.Errorf("%s", stderr.String())
	}

	return nil
}

func (c *Kubectl) Preflight(preflightURI string) error {
	args := []string{
		"preflight",
		preflightURI,
	}

	cmd := c.kubectlCommand(args...)
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
		fmt.Printf("error running kubectl preflight: \n stderr %s\n stdout %s\n", bytes.Join(stderr, []byte("\n")), bytes.Join(stdout, []byte("\n")))
		return errors.Wrap(err, "failed to run kubectl preflight")
	}

	return nil
}

func (c *Kubectl) Apply(namespace string, yamlDoc []byte, dryRun bool) error {
	args := []string{
		"apply",
	}

	if dryRun {
		args = append(args, "--dry-run")
	}

	targetNamespace := os.Getenv("DEFAULT_NAMESPACE")
	if namespace != "." {
		targetNamespace = namespace
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
		fmt.Printf("error running kubectl apply: \n stderr %s\n stdout %s\n", bytes.Join(stderr, []byte("\n")), bytes.Join(stdout, []byte("\n")))
		return errors.Wrap(err, "failed to run kubectl apply")
	}

	return nil
}

func (c *Kubectl) kubectlCommand(args ...string) *exec.Cmd {
	return exec.Command(c.exe, append(c.connectArgs(), args...)...)
}
