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
	"time"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
)

const systemdServiceFile = `# Copyright The containerd Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

[Unit]
Description=containerd container runtime
Documentation=https://containerd.io
After=network.target local-fs.target

[Service]
ExecStartPre=-/sbin/modprobe overlay
ExecStart=/var/lib/containerd/bin/containerd

Type=notify
Delegate=yes
KillMode=process
Restart=always
RestartSec=5
# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNPROC=infinity
LimitCORE=infinity
LimitNOFILE=infinity
# Comment TasksMax if your systemd version does not supports it.
# Only systemd 226 and above support this version.
TasksMax=infinity
OOMScoreAdjust=-999

[Install]
WantedBy=multi-user.target

`

func verifyContainerdInstallation() error {
	currentStatus, err := getCurrentStatus()
	if err != nil {
		return errors.Wrap(err, "get current status")
	}

	if currentStatus == "active\n" {
		// TODO versions?
		return nil
	}

	// verify that we have permission to install
	prompt := promptui.Prompt{
		Label:     "containerd is not found and required to run the application. Press Y to install containerd on this server and continue, or press n to exit.",
		IsConfirm: true,
	}

	_, err = prompt.Run()
	if err != nil {
		return errors.New("Cannot continue without installing containerd. You can install runc manually or try again and allow KOTS to install containerd.")
	}

	packageURI := `https://github.com/containerd/containerd/releases/download/v1.5.1/containerd-1.5.1-linux-amd64.tar.gz`
	resp, err := http.Get(packageURI)
	if err != nil {
		return errors.Wrap(err, "download containerd")
	}
	defer resp.Body.Close()

	// TODO install runc

	// extract containerd into a new directory
	installDir := `/var/lib/containerd/`
	if _, err := os.Stat(installDir); err == nil {
		if err := os.RemoveAll(installDir); err != nil {
			return errors.Wrap(err, "remove previous containerd")
		}
	}

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return errors.Wrap(err, "mkdir")
	}

	if err := extractArchiveStreamToDir(resp.Body, installDir); err != nil {
		return errors.Wrap(err, "extract")
	}

	if err := generateDefaultConfig(); err != nil {
		return errors.Wrap(err, "generate default config")
	}

	if err := installContainerdService(); err != nil {
		return errors.Wrap(err, "install containerd service")
	}

	return nil
}

func getCurrentStatus() (string, error) {
	// TODO check runc
	cmd := exec.Command("systemctl", "check", "containerd")
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

func extractArchiveStreamToDir(r io.ReadCloser, dest string) error {
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
			if err := os.Mkdir(filepath.Join(dest, (header.Name)), 0755); err != nil {
				return errors.Wrap(err, "mkdir")
			}
		case tar.TypeReg:
			outFile, err := os.Create(filepath.Join(dest, header.Name))
			if err != nil {
				return errors.Wrap(err, "create")
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrap(err, "copy")
			}
			if err := os.Chmod(filepath.Join(dest, header.Name), fs.FileMode(header.Mode)); err != nil {
				return errors.Wrap(err, "chmod")
			}

			outFile.Close()

		default:
			return errors.New("unknown type")
		}

	}

	return nil
}

func generateDefaultConfig() error {
	cmd := exec.Command("/var/lib/containerd/bin/containerd", "config", "default")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "exec containerd config default")
	}

	configFile := `/etc/containerd/config.toml`
	if _, err := os.Stat(configFile); err == nil {
		if err := os.RemoveAll(configFile); err != nil {
			return errors.Wrap(err, "remove containerd config")
		}
	}

	d := filepath.Dir(configFile)
	if _, err := os.Stat(d); os.IsNotExist(err) {
		if err := os.MkdirAll(d, 0755); err != nil {
			return errors.Wrap(err, "mkdir")
		}
	}

	if err := ioutil.WriteFile(configFile, out, 0644); err != nil {
		return errors.Wrap(err, "write containerd config")
	}

	return nil
}

func installContainerdService() error {
	serviceFile := `/etc/systemd/system/containerd.service`
	if _, err := os.Stat(serviceFile); err == nil {
		if err := os.RemoveAll(serviceFile); err != nil {
			return errors.Wrap(err, "remove previous service file")
		}
	}

	if err := ioutil.WriteFile(serviceFile, []byte(systemdServiceFile), 0644); err != nil {
		return errors.Wrap(err, "write systemd service file")
	}

	cmd := exec.Command("systemctl", "enable", "containerd")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", out)
		return errors.Wrap(err, "enable service")
	}

	cmd = exec.Command("systemctl", "start", "containerd")
	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", out)
		return errors.Wrap(err, "start service")
	}

	giveUpAfter := time.Now().Add(time.Second * 15)

	fmt.Println("waiting for containerd service to become active")
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
