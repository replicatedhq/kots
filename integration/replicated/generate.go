package replicated

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/archiveutil"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/util"
)

// GenerateTest will create a new replicated app fixture for integration tests
func GenerateTest(name string, rawArchivePath string) error {
	namespace := "test_ns"
	integrationLicenseData := fmt.Sprintf(`apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: integration
spec:
  licenseID: %s
  appSlug: %s
  endpoint: http://localhost:3001
  signature: IA==
`, name, name)

	replicatedAppArchive, err := generateReplicatedAppArchive(rawArchivePath)
	if err != nil {
		return errors.Wrap(err, "failed to generate replicated app archive")
	}

	expectedFilesystem, err := generateExpectedFilesystem(namespace, rawArchivePath)
	if err != nil {
		return errors.Wrap(err, "failed to generate expected filesystem")
	}

	testRoot := filepath.Join("integration", "replicated", "tests", name)
	if err := os.MkdirAll(testRoot, 0755); err != nil {
		return errors.Wrap(err, "failed to create test root")
	}

	err = os.WriteFile(filepath.Join(testRoot, "license.yaml"), []byte(integrationLicenseData), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write license")
	}

	err = os.WriteFile(filepath.Join(testRoot, "archive.tar.gz"), replicatedAppArchive, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write archive")
	}

	tempExpectedFile, err := os.MkdirTemp("", "kotsintegration")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	defer os.RemoveAll(tempExpectedFile)

	err = os.WriteFile(filepath.Join(tempExpectedFile, "archive.tar.gz"), expectedFilesystem, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write to temp file")
	}

	expectedRoot := filepath.Join(testRoot, "expected")

	err = util.ExtractTGZArchive(filepath.Join(tempExpectedFile, "archive.tar.gz"), expectedRoot)
	if err != nil {
		return errors.Wrapf(err, "failed to extract archive to %s", expectedRoot)
	}

	return nil
}

func generateReplicatedAppArchive(rawArchivePath string) ([]byte, error) {
	archiveDir, err := os.MkdirTemp("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	archiveFile := filepath.Join(archiveDir, "archive.tar.gz")

	filenames := map[string]string{rawArchivePath: ""}
	if err := archiveutil.CreateTGZ(context.TODO(), filenames, archiveFile); err != nil {
		return nil, errors.Wrap(err, "failed to create archive")
	}
	b, err := os.ReadFile(archiveFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive file")
	}

	return b, nil
}

// generateExpectedFilesystem uses kots to simularae a pull from a local source
// and then creates a tar from what the output is. because of this, it's expected
// that kots is working as expected when creating a new test
func generateExpectedFilesystem(namespace, rawArchivePath string) ([]byte, error) {
	tmpRootDir, err := os.MkdirTemp("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tmpRootDir)

	pullOptions := pull.PullOptions{
		RootDir:             tmpRootDir,
		LocalPath:           rawArchivePath,
		Namespace:           namespace,
		ExcludeKotsKinds:    true,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
		Silent:              true,
	}

	_, err = pull.Pull("replicated://integration", pullOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pull")
	}

	archiveDir, err := os.MkdirTemp("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	archiveFile := filepath.Join(archiveDir, "expected.tar.gz")

	filenames := map[string]string{
		filepath.Join(tmpRootDir, "upstream"): "",
		filepath.Join(tmpRootDir, "base"):     "",
		filepath.Join(tmpRootDir, "overlays"): "",
	}
	skippedFilesPath := filepath.Join(tmpRootDir, "skippedFiles")
	if _, err := os.Stat(skippedFilesPath); err == nil {
		filenames[skippedFilesPath] = ""
	}
	if err := archiveutil.CreateTGZ(context.TODO(), filenames, archiveFile); err != nil {
		return nil, errors.Wrap(err, "failed to create archive")
	}
	b, err := os.ReadFile(archiveFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive file")
	}

	return b, nil
}
