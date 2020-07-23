package upload

import (
	"os"
	"path/filepath"

	"github.com/mholt/archiver"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

// GenerateTest will create a new upload test fixture for integration tests
func GenerateTest(name string, applicationPath string) error {
	testRoot := filepath.Join("integration", "upload", "tests", name)
	if err := os.MkdirAll(testRoot, 0755); err != nil {
		return errors.Wrap(err, "failed to create test root")
	}

	inputRoot := filepath.Join(testRoot, "input")
	if err := os.MkdirAll(inputRoot, 0755); err != nil {
		return errors.Wrap(err, "failed to create input root")
	}

	err := copy.Copy(applicationPath, inputRoot)
	if err != nil {
		return errors.Wrap(err, "failed to copy input")
	}

	// Create the expected archive
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}

	paths := []string{
		filepath.Join(applicationPath, "upstream"),
		filepath.Join(applicationPath, "base"),
		filepath.Join(applicationPath, "overlays"),
		filepath.Join(applicationPath, "errors"),
	}

	if err := tarGz.Archive(paths, filepath.Join(testRoot, "expected-archive.tar.gz")); err != nil {
		return errors.Wrap(err, "failed to create tar gz")
	}

	return nil
}
