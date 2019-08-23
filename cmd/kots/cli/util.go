package cli

import (
	"os"
	"path"
	"strings"
)

func ExpandDir(input string) string {
	if !strings.HasPrefix(input, "~") {
		return input
	}

	return path.Join(homeDir(), input[1:])
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}
