package cluster

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
)

func verifyRuncInstallation() error {
	currentVersion, err := getCurrentRuncVersion()
	if err != nil {
		return errors.Wrap(err, "get current version")
	}

	if currentVersion != "" {
		return nil
	}

	// verify that we have permission to install
	prompt := promptui.Prompt{
		Label:     "runc is not found and required to run the application. Press Y to install runc on this server and continue, or press n to exit.",
		IsConfirm: true,
	}

	_, err = prompt.Run()
	if err != nil {
		return errors.New("Cannot continue without installing runc. You can install runc manually or try again and allow KOTS to install runc.")
	}

	packageURI := `https://github.com/opencontainers/runc/releases/download/v1.0.0-rc95/runc.amd64`
	resp, err := http.Get(packageURI)
	if err != nil {
		return errors.Wrap(err, "download runc")
	}
	defer resp.Body.Close()

	installDir := `/usr/bin/`
	out, err := os.Create(filepath.Join(installDir, "runc"))
	if err != nil {
		return errors.Wrap(err, "create file")
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return errors.Wrap(err, "copy response")
	}
	out.Close()

	if err := os.Chmod(filepath.Join(installDir, "runc"), 0755); err != nil {
		return errors.Wrap(err, "chmod")
	}

	return nil
}

func getCurrentRuncVersion() (string, error) {
	// TODO check runc
	cmd := exec.Command("runc", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", nil
	}

	return string(out), nil
}
