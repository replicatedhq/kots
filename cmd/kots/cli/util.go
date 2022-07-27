package cli

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
)

func ExpandDir(input string) string {
	if input == "" {
		return ""
	}

	if strings.HasPrefix(input, "~") {
		input = filepath.Join(util.HomeDir(), strings.TrimPrefix(input, "~"))
	}

	uploadPath, err := filepath.Abs(input)
	if err != nil {
		panic(errors.Wrapf(err, "unable to expand %q to absolute path", input))
	}

	return uploadPath
}
