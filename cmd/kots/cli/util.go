package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	kubernetesConfigFlags *genericclioptions.ConfigFlags
)

func ExpandDir(input string) string {
	uploadPath, err := filepath.Abs(input)
	if err != nil {
		panic(errors.Wrapf(err, "unable to expand %q to absolute path", input))
	}

	if !strings.HasPrefix(uploadPath, "~") {
		return uploadPath
	}

	return filepath.Join(homeDir(), uploadPath[1:])
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}
