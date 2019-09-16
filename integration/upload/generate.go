package upload

import (
	"os"
	"path"

	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

// GenerateTest will create a new upload test fixture for integration tests
func GenerateTest(name string, applicationPath string) error {
	testRoot := path.Join("integration", "upload", "tests", name)
	if err := os.MkdirAll(testRoot, 0755); err != nil {
		return errors.Wrap(err, "failed to create test root")
	}

	inputRoot := path.Join(testRoot, "input")
	if err := os.MkdirAll(inputRoot, 0755); err != nil {
		return errors.Wrap(err, "failed to create input root")
	}

	err := copy.Copy(applicationPath, inputRoot)
	if err != nil {
		return errors.Wrap(err, "failed to copy input")
	}

	// TODO...

	return nil
}
