package binaries

import (
	"io/fs"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
)

var (
	kustomizeBinPath string
)

// InitKustomize will discover kustomize binary path from the environment
func InitKustomize() (err error) {
	kustomizeBinPath, err = discoverKustomizeBinPath(os.DirFS("/"))
	if err != nil {
		return errors.Wrap(err, "discover kustomize bin path")
	}
	logger.Infof("Found kustomize binary at %s", kustomizeBinPath)
	return nil
}

// GetKustomizeBinPath returns the path to the kustomize binary
func GetKustomizeBinPath() string {
	if kustomizeBinPath != "" {
		return kustomizeBinPath
	}

	return "kustomize"
}

func discoverKustomizeBinPath(fileSystem fs.FS) (string, error) {
	// NOTE: exec.LookPath does not yet support io/fs
	binPath, err := exec.LookPath("kustomize")
	if err != nil {
		return "", errors.Wrap(err, "look path")
	}

	return binPath, nil
}
