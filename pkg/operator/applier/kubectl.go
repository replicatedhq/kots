package applier

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/marccampbell/yaml-toolbox/pkg/splitter"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kustomize"
	"github.com/replicatedhq/kots/pkg/logger"
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
	tmp, err := ioutil.TempDir("", "kots-apply-")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create temp directory")
	}
	defer os.RemoveAll(tmp)

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

// ApplyCreateOrPatch attempts to run a `kubectl apply` on the yaml document. If it fails
func (c *Kubectl) ApplyCreateOrPatch(targetNamespace string, slug string, yamlDoc []byte, dryRun bool, wait bool, annotateSlug bool) ([]byte, []byte, error) {

	stdout, stderr, err := c.Apply(targetNamespace, slug, yamlDoc, dryRun, wait, annotateSlug)
	if err == nil {
		return stdout, stderr, nil
	} else if !strings.Contains(string(stderr), "metadata.annotations: Too long") {
		return stdout, stderr, errors.Wrap(err, "failed attempted kubectl apply")
	}
	logger.Info("Failed to apply document: metadata too long. Trying to 'create' or 'patch' instead")

	docs, err := splitter.SplitYAML(yamlDoc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "split yaml")
	}

	var combinedStdout, combinedStderr []byte
	for name, b := range docs {
		stdout, stderr, err = c.createOrPatchSingleDoc(name, targetNamespace, slug, b, dryRun, wait, annotateSlug)
		combinedStdout = append(combinedStdout, stdout...)
		combinedStderr = append(combinedStderr, stderr...)

		if err != nil {
			return combinedStdout, combinedStderr, errors.Wrapf(err, "failed to run kubectl create/patch for document %s", name)
		}
	}

	return combinedStdout, combinedStderr, nil
}

func (c *Kubectl) createOrPatchSingleDoc(name string, targetNamespace string, slug string, yamlDoc []byte, dryRun bool, wait bool, annotateSlug bool) ([]byte, []byte, error) {
	tmp, err := ioutil.TempDir("", "kots-create-")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create temp directory")
	}
	defer os.RemoveAll(tmp)

	docPath := filepath.Join(tmp, "doc.yaml")
	if err := ioutil.WriteFile(docPath, yamlDoc, 0644); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to write %s", docPath)
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

	renderedManifest, err := exec.Command(kustomize.GetKustomizePath(kotsutil.KotsKustomizeVersion), "build", tmp).Output()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to run kustomize build")
	}

	renderedManifestPath := filepath.Join(tmp, "patch.yaml")
	if err := ioutil.WriteFile(renderedManifestPath, renderedManifest, 0644); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to write %s", renderedManifestPath)
	}
	// Try to create and checkout output to see if it already exists
	cmd := c.kubectlCreateCommand(renderedManifestPath, targetNamespace, dryRun, wait)

	stdout, stderr, err := Run(cmd)
	if err == nil {
		return stdout, stderr, nil
	} else if !strings.Contains(string(stderr), "already exists") {
		return stdout, stderr, errors.Wrap(err, "failed attempted kubectl create")
	}
	logger.Infof("Failed to create document %s: already exists. Trying to 'patch' instead", name)

	cmd = c.kubectlPatchCommand(renderedManifestPath, targetNamespace, dryRun, wait)

	stdout, stderr, err = Run(cmd)
	return stdout, stderr, errors.Wrap(err, "failed to run kubectl patch")
}

func (c *Kubectl) kubectlCreateCommand(renderedManifestPath string, targetNamespace string, dryRun bool, wait bool) *exec.Cmd {

	args := []string{
		"create",
		fmt.Sprintf("-f=%s", renderedManifestPath),
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

	return c.kubectlCommand(args...)
}

func (c *Kubectl) kubectlPatchCommand(renderedManifestPath string, targetNamespace string, dryRun bool, wait bool) *exec.Cmd {

	args := []string{
		"patch",
		fmt.Sprintf("--patch-file=%s", renderedManifestPath),
		fmt.Sprintf("-f=%s", renderedManifestPath),
		"--type=merge",
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

	return c.kubectlCommand(args...)
}

func (c *Kubectl) kubectlCommand(args ...string) *exec.Cmd {
	return exec.Command(c.kubectl, append(args, c.connectArgs()...)...)
}
