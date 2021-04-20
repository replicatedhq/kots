package applier

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	rest "k8s.io/client-go/rest"
)

type Kubectl struct {
	kubectl string
	config  *rest.Config
}

func NewKubectl(kubectl string, config *rest.Config) *Kubectl {
	return &Kubectl{
		kubectl: kubectl,
		config:  config,
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

func (c *Kubectl) Remove(targetNamespace string, yamlDoc []byte, wait bool) ([]byte, []byte, error) {
	args := []string{
		"delete",
		fmt.Sprintf("--wait=%t", wait),
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

func (c *Kubectl) Apply(targetNamespace string, slug string, yamlDoc []byte, dryRun bool, wait bool, annotateSlug bool) ([]byte, []byte, error) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create temp directory")
	}
	defer os.Remove(tmp)

	args := []string{
		"apply",
		"--kustomize",
		tmp,
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

	yamlPath := filepath.Join(tmp, "doc.yaml")
	if err := ioutil.WriteFile(yamlPath, yamlDoc, 0644); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to write %s", yamlPath)
	}

	kustomizationPath := filepath.Join(tmp, "kustomization.yaml")
	kustomizationYaml := `
resources:
- doc.yaml
`
	if annotateSlug {
		kustomizationYaml = fmt.Sprintf(`
resources:
- doc.yaml

commonAnnotations:
  kots.io/app-slug: %q
`, slug)
	}
	if err := ioutil.WriteFile(kustomizationPath, []byte(kustomizationYaml), 0644); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to write %s", kustomizationPath)
	}

	cmd := c.kubectlCommand(args...)

	stdout, stderr, err := Run(cmd)
	return stdout, stderr, errors.Wrap(err, "failed to run kubectl apply")
}

func (c *Kubectl) kubectlCommand(args ...string) *exec.Cmd {
	return exec.Command(c.kubectl, append(args, c.connectArgs()...)...)
}
