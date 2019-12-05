package cli

import (
	"os"
	"path/filepath"
	"strings"
)

func ExpandDir(input string) string {
	if !strings.HasPrefix(input, "~") {
		return input
	}

	return filepath.Join(homeDir(), input[1:])
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

func defaultKubeConfig() string {
	if len(os.Getenv("KUBECONFIG")) > 0 {
		return os.Getenv("KUBECONFIG")
	}
	return filepath.Join(homeDir(), ".kube", "config")
}
