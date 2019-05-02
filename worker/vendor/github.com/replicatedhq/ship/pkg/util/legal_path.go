package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// IsLegalPath checks if the provided path is a relative path within the current working directory or within the os tempdir.
// If it is not, it returns an error.
func IsLegalPath(path string) error {

	if filepath.IsAbs(path) {
		relAbsPath, err := filepath.Rel(os.TempDir(), path)
		if err != nil {
			return fmt.Errorf("cannot write to an absolute path: %s, got error finding relative path from tempdir: %s", path, err.Error())
		}

		// subdirectories of the os tempdir are fine
		if !strings.Contains(relAbsPath, "..") {
			return nil
		}

		return fmt.Errorf("cannot write to an absolute path: %s", path)
	}

	relPath, err := filepath.Rel(".", path)
	if err != nil {
		return errors.Wrap(err, "find relative path to dest")
	}

	if strings.Contains(relPath, "..") {
		return fmt.Errorf("cannot write to a path that is a parent of the working dir: %s", relPath)
	}

	return nil
}
