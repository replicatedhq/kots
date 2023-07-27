package replicated

import (
	"fmt"
	"os"
	"path"

	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/pull"
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

	testRoot := path.Join("integration", "replicated", "tests", name)
	if err := os.MkdirAll(testRoot, 0755); err != nil {
		return errors.Wrap(err, "failed to create test root")
	}

	err = os.WriteFile(path.Join(testRoot, "license.yaml"), []byte(integrationLicenseData), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write license")
	}

	err = os.WriteFile(path.Join(testRoot, "archive.tar.gz"), replicatedAppArchive, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write archive")
	}

	expectedRoot := path.Join(testRoot, "expected")
	if err := os.MkdirAll(expectedRoot, 0755); err != nil {
		return errors.Wrap(err, "failed to create expected root")
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	tempExpectedFile, err := os.MkdirTemp("", "kotsintegration")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	defer os.RemoveAll(tempExpectedFile)
	err = os.WriteFile(path.Join(tempExpectedFile, "archive.tar.gz"), expectedFilesystem, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write to temp file")
	}
	if err := tarGz.Unarchive(path.Join(tempExpectedFile, "archive.tar.gz"), expectedRoot); err != nil {
		return errors.Wrap(err, "failed to unarchive expected")
	}

	return nil
}

func generateReplicatedAppArchive(rawArchivePath string) ([]byte, error) {
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}

	archiveDir, err := os.MkdirTemp("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	archiveFile := path.Join(archiveDir, "archive.tar.gz")
	if err := tarGz.Archive([]string{rawArchivePath}, archiveFile); err != nil {
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

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}

	archiveDir, err := os.MkdirTemp("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	archiveFile := path.Join(archiveDir, "expected.tar.gz")

	paths := []string{
		path.Join(tmpRootDir, "upstream"),
		path.Join(tmpRootDir, "base"),
		path.Join(tmpRootDir, "overlays"),
	}
	skippedFilesPath := path.Join(tmpRootDir, "skippedFiles")
	if _, err := os.Stat(skippedFilesPath); err == nil {
		paths = append(paths, skippedFilesPath)
	}
	if err := tarGz.Archive(paths, archiveFile); err != nil {
		return nil, errors.Wrap(err, "failed to create archive")
	}
	b, err := os.ReadFile(archiveFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive file")
	}

	return b, nil
}
