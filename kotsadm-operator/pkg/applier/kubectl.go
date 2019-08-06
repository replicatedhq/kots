package applier

import (
	"bytes"
	"fmt"
	"os/exec"

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

func (c *Kubectl) Remove(yamlDoc []byte) error {
	args := []string{
		"delete",
		"-n",
		"default",
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
		return fmt.Errorf("%s", stderr.String())
	}

	return nil
}

func (c *Kubectl) Apply(yamlDoc []byte) error {
	args := []string{
		"apply",
		"-n",
		"default",
		"-f",
		"-",
	}
	cmd := c.kubectlCommand(args...)
	cmd.Stdin = bytes.NewReader(yamlDoc)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout

	if err := cmd.Run(); err != nil {
		fmt.Printf("error running kubectl apply: %q\n", err)
		return fmt.Errorf("%s", stderr.String())
	}

	return nil
}

func (c *Kubectl) kubectlCommand(args ...string) *exec.Cmd {
	return exec.Command(c.exe, append(c.connectArgs(), args...)...)
}
