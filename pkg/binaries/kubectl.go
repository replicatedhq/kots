package binaries

import (
	"io/fs"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
)

var (
	kubectlBinPath string
)

// InitKubectl will discover kubectl binary path from the environment
func InitKubectl() (err error) {
	kubectlBinPath, err = discoverKubectlBinPath(os.DirFS("/"))
	if err != nil {
		return errors.Wrap(err, "discover kubectl bin path")
	}
	logger.Infof("Found kubectl binary at %s", kubectlBinPath)
	return nil
}

// GetKubectlBinPath returns the path to the kubectl binary
func GetKubectlBinPath() string {
	if kubectlBinPath != "" {
		return kubectlBinPath
	}

	return "kubectl"
}

func discoverKubectlBinPath(fileSystem fs.FS) (string, error) {
	// NOTE: exec.LookPath does not yet support io/fs
	binPath, err := exec.LookPath("kubectl")
	if err != nil {
		return "", errors.Wrap(err, "look path")
	}

	return binPath, nil
}
