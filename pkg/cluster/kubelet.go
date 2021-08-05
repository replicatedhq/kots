package cluster

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const kubeletConfig = `apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
evictionHard:
  memory.available:  "200Mi"
`

const kubeletServiceFileWithNodeAuthorizer = `[Service]
Environment="KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf --kubeconfig=/etc/kubernetes/kubelet.conf"
Environment="KUBELET_CONFIG_ARGS=--config=/var/lib/kubelet/config.yaml"
Environment="KUBELET_EXTRA_ARGS=--container-runtime=remote --runtime-request-timeout=15m --container-runtime-endpoint=unix:///run/containerd/containerd.sock"
EnvironmentFile=-/var/lib/kubelet/kubeadm-flags.env
EnvironmentFile=-/etc/default/kubelet
ExecStart=
ExecStart=/usr/bin/kubelet $KUBELET_KUBECONFIG_ARGS $KUBELET_CONFIG_ARGS $KUBELET_KUBEADM_ARGS $KUBELET_EXTRA_ARGS
`
const kubeletServiceFileWithoutNodeAuthorizer = `[Service]
Environment="KUBELET_KUBECONFIG_ARGS=--kubeconfig=/etc/kubernetes/kubelet.conf"
Environment="KUBELET_CONFIG_ARGS=--config=/var/lib/kubelet/config.yaml"
Environment="KUBELET_EXTRA_ARGS=--container-runtime=remote --runtime-request-timeout=15m --container-runtime-endpoint=unix:///run/containerd/containerd.sock"
EnvironmentFile=-/var/lib/kubelet/kubeadm-flags.env
EnvironmentFile=-/etc/default/kubelet
ExecStart=
ExecStart=/usr/bin/kubelet $KUBELET_KUBECONFIG_ARGS $KUBELET_CONFIG_ARGS $KUBELET_KUBEADM_ARGS $KUBELET_EXTRA_ARGS
`

func installKubelet(useNodeAuthorizer bool) error {
	currentStatus, err := getCurrentKubeletStatus()
	if err != nil {
		return errors.Wrap(err, "get current status")
	}

	if currentStatus == "active\n" {
		// TODO versions?
		return nil
	}

	packageURI := `https://dl.k8s.io/v1.21.1/kubernetes-server-linux-amd64.tar.gz`
	resp, err := http.Get(packageURI)
	if err != nil {
		return errors.Wrap(err, "download kubelet")
	}
	defer resp.Body.Close()

	// extract kubelet to a new directory
	if err := extractOneFileFromArchiveStreamToDir("kubelet", resp.Body, "/usr/bin"); err != nil {
		return errors.Wrap(err, "extract one file")
	}

	if err := writeKubeletConfig(); err != nil {
		return errors.Wrap(err, "write kubelet config")
	}

	if err := installKubeletService(useNodeAuthorizer); err != nil {
		return errors.Wrap(err, "install kubelet service")
	}

	return nil
}

func getCurrentKubeletStatus() (string, error) {
	cmd := exec.Command("systemctl", "check", "kubelet")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if string(out) != "active\n" {
				return string(out), nil
			}

			return "", fmt.Errorf("exit error running systemctl: %s", (exitErr.Error()))
		}

		return "", errors.Wrap(err, "run systemctl")
	}

	return string(out), nil
}

func extractOneFileFromArchiveStreamToDir(filename string, r io.ReadCloser, dest string) error {
	uncompressedStream, err := gzip.NewReader(r)
	if err != nil {
		return errors.Wrap(err, "create gzip reader")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return errors.Wrap(err, "read next")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			if !strings.HasSuffix(header.Name, filename) {
				continue
			}
			outFile, err := os.Create(filepath.Join(dest, filename))
			if err != nil {
				return errors.Wrap(err, "create")
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrap(err, "copy")
			}
			if err := os.Chmod(filepath.Join(dest, filename), fs.FileMode(header.Mode)); err != nil {
				return errors.Wrap(err, "chmod")
			}

			outFile.Close()

		default:
			return errors.New("unknown type")
		}

	}

	return nil
}

func writeKubeletConfig() error {
	dir := `/var/lib/kubelet`
	if _, err := os.Stat(dir); err == nil {
		if err := os.RemoveAll(dir); err != nil {
			return errors.Wrap(err, "remove previous kubelet dir")
		}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, "mkdir")
	}

	if err := ioutil.WriteFile(filepath.Join(dir, "config.yaml"), []byte(kubeletConfig), 0644); err != nil {
		return errors.Wrap(err, "write kubelet config")
	}

	return nil
}

func installKubeletService(useNodeAuthorizer bool) error {
	serviceFile := `/etc/systemd/system/kubelet.service`
	if _, err := os.Stat(serviceFile); err == nil {
		if err := os.RemoveAll(serviceFile); err != nil {
			return errors.Wrap(err, "remove previous service file")
		}
	}

	contents := kubeletServiceFileWithoutNodeAuthorizer
	if useNodeAuthorizer {
		contents = kubeletServiceFileWithNodeAuthorizer
	}

	if err := ioutil.WriteFile(serviceFile, []byte(contents), 0644); err != nil {
		return errors.Wrap(err, "write systemd service file")
	}

	cmd := exec.Command("systemctl", "enable", "kubelet")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", out)
		return errors.Wrap(err, "enable service")
	}

	cmd = exec.Command("systemctl", "start", "kubelet")
	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", out)
		return errors.Wrap(err, "start service")
	}

	giveUpAfter := time.Now().Add(time.Second * 15)

	fmt.Println("waiting for kubelet service to become active")
	for {
		if time.Now().After(giveUpAfter) {
			return errors.New("systemd service did not become active")
		}

		afterStatus, afterErr := getCurrentStatus()
		if afterErr == nil && afterStatus == "active\n" {
			return nil
		}

		fmt.Printf("%q", afterStatus)
		time.Sleep(time.Second)
	}
}
