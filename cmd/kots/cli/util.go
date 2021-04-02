package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func ExpandDir(input string) string {
	if input == "" {
		return ""
	}

	if strings.HasPrefix(input, "~") {
		input = filepath.Join(homeDir(), strings.TrimPrefix(input, "~"))
	}

	uploadPath, err := filepath.Abs(input)
	if err != nil {
		panic(errors.Wrapf(err, "unable to expand %q to absolute path", input))
	}

	return uploadPath
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}
