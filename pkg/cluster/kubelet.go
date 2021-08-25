package cluster

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	kubeletConfig = `apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
evictionHard:
  memory.available:  "200Mi"
featureGates:
  KubeletInUserNamespace: true  
`
	kubeletKubeconfigFilename = "kubelet-kubeconfig.yaml"
	kubeletConfigFilename     = "config.yaml"
)

// startKubelet will (spwan) a kubelet process that's basically unsupervised
// we need to think about resiliency and supervision for this kubelet before this can be reliable
// we do this instead of a systemd service for kubelet because we don't want to ask for sudo/root
func startKubelet(ctx context.Context, dataDir string) error {
	// make a kubelet directory under datadir
	kubeletDir := filepath.Join(dataDir, "kubelet")
	if _, err := os.Stat(kubeletDir); os.IsNotExist(err) {
		if err := os.MkdirAll(kubeletDir, 0755); err != nil {
			return errors.Wrap(err, "mkdir kubelet")
		}
	}

	// TODO check the version of kubelet in case there was an old version

	// write the kubelet config file
	if err := writeKubeletConfig(kubeletDir); err != nil {
		return errors.Wrap(err, "write kubelet config")
	}

	// write the kubelet kubeconfig file
	if err := writeKubeletKubeconfig(dataDir, kubeletDir); err != nil {
		return errors.Wrap(err, "write kubelet kubeconfig")
	}

	// spawn the kubelet process
	if err := spawnKubelet(dataDir, kubeletDir); err != nil {
		return errors.Wrap(err, "spawn kubelet")
	}

	return nil
}

func spawnKubelet(dataDir string, kubeletDir string) error {
	go func() {
		args := []string{
			fmt.Sprintf("--kubeconfig=%s", filepath.Join(kubeletDir, kubeletKubeconfigFilename)),
			fmt.Sprintf("--config=%s", filepath.Join(kubeletDir, kubeletConfigFilename)),
			"--container-runtime=remote",
			fmt.Sprintf("--container-runtime-endpoint=unix://%s/containerd/containerd.sock", dataDir),
			fmt.Sprintf("--root-dir=%s", kubeletDir),
			fmt.Sprintf("--cert-dir=%s", filepath.Join(kubeletDir, "pki")),
		}

		cmd := exec.Command(filepath.Join(BinRoot(dataDir), "kubelet"), args...)
		cmd.Env = os.Environ() // TODO

		// TODO stream the output of stdout and stderr to files

		// TODO stream the output of stdout and stderr to files
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			fmt.Printf("%s\n", stderr.String())
			panic(err)
		}

		fmt.Printf("%s\n", stdout.String())

	}()

	return nil
}

// writeKubeletKubeconfig will write a file named kubelet-kubeconfig.yaml
// to the kubelet dir. this is the kubeconfig that the kubelet will use to
// connect to the api server. it's not using a bootstrap token yet, this is
// using a static token that we JIT provision when writing this file
func writeKubeletKubeconfig(dataDir string, kubeletDir string) error {
	b, err := getKubeletKubeconfig(dataDir)
	if err != nil {
		return errors.Wrap(err, "get kubelet bootstrap config")
	}

	if err := ioutil.WriteFile(filepath.Join(kubeletDir, kubeletKubeconfigFilename), b, 0644); err != nil {
		return errors.Wrap(err, "write kubelet kubeconfig")
	}

	return nil
}

// getKubeletKubeconfig will return the kubeconfig used by the kubelet
func getKubeletKubeconfig(dataDir string) ([]byte, error) {
	certFile, err := caCertFilePath(dataDir)
	if err != nil {
		return nil, errors.Wrap(err, "get cert file path")
	}
	data, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, errors.Wrap(err, "read cert file")
	}
	encodedCert := base64.StdEncoding.EncodeToString(data)

	bootstrapToken := "NOT_VALID" // TODO

	b := fmt.Sprintf(`apiVersion: v1
clusters:
- name: kubernetes
  cluster:
    certificate-authority-data: %s
    server: "https://localhost:8443"

contexts:
- name: tls-bootstrap-token-user@kubernetes
  context:
    cluster: kubernetes
    user: tls-bootstrap-token-user

current-context: tls-bootstrap-token-user@kubernetes
kind: Config
preferences: {}
users:
- name: tls-bootstrap-token-user
  user:
    token: %s`, encodedCert, bootstrapToken)

	return []byte(b), nil
}

// writeKubeletConfig writes the current kubelet config to a file to pass in
// when spawning the kubelet process.
func writeKubeletConfig(kubeletDir string) error {
	if err := ioutil.WriteFile(filepath.Join(kubeletDir, kubeletConfigFilename), []byte(kubeletConfig), 0644); err != nil {
		return errors.Wrap(err, "write kubelet config")
	}

	return nil
}
