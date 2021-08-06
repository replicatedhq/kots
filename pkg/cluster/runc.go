package cluster

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

// verifyRunc will ensure that a compatble version of runc is present in the runc directory
func verifyRunc(dataDir string) error {
	installDir := filepath.Join(dataDir, "runc")

	if _, err := os.Stat(installDir); os.IsNotExist(err) {
		if err := os.MkdirAll(installDir, 0755); err != nil {
			return errors.Wrapf(err, "create install dir")
		}
	}

	currentVersion, err := getCurrentRuncVersion(installDir)
	if err != nil {
		return errors.Wrap(err, "get current version")
	}

	// TODO which versions do we support?
	if currentVersion != "" {
		return nil
	}

	packageURI := `https://github.com/opencontainers/runc/releases/download/v1.0.0-rc95/runc.amd64`
	resp, err := http.Get(packageURI)
	if err != nil {
		return errors.Wrap(err, "download runc")
	}
	defer resp.Body.Close()

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

func getCurrentRuncVersion(dir string) (string, error) {
	// TODO check runc
	cmd := exec.Command(filepath.Join(dir, "runc"), "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", nil
	}

	return string(out), nil
}
