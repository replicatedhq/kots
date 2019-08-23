package cli

import (
	"os"
	"strings"
)

func ExpandDir(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	return homeDir() + path[1:]
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}
