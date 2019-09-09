package pull

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"text/template"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
)

func GenerateTest(name string, rawArchivePath string) error {
	integrationLicenseData := fmt.Sprintf(`apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: integration
spec:
  licenseID: %s
  appSlug: %s
  endpoint: http://localhost:3000
  signature: IA==
`, name, name)

	replicatedAppArchive, err := generateReplicatedAppArchive(rawArchivePath)
	if err != nil {
		return errors.Wrap(err, "failed to generate replicated app archive")
	}

	expectedFilesystem, err := generateExpectedFilesystem(rawArchivePath)
	if err != nil {
		return errors.Wrap(err, "failed to generate expected filesystem")
	}

	context := map[string]string{
		"Name":                 name,
		"LicenseData":          integrationLicenseData,
		"ReplicatedAppArchive": base64.StdEncoding.EncodeToString(replicatedAppArchive),
		"ExpectedFilesystem":   base64.StdEncoding.EncodeToString(expectedFilesystem),
	}

	tmpl, err := template.New(name).Parse(testTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to parse template")
	}

	f, err := os.Create(path.Join("integration", "replicated", "pull", fmt.Sprintf("%s.go", name)))
	if err != nil {
		return errors.Wrap(err, "failed to create new test file")
	}

	if err := tmpl.Execute(f, context); err != nil {
		return errors.Wrap(err, "failed to execute template")
	}

	return nil
}

func generateReplicatedAppArchive(rawArchivePath string) ([]byte, error) {
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}

	archiveDir, err := ioutil.TempDir("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	archiveFile := path.Join(archiveDir, "archive.tar.gz")
	if err := tarGz.Archive([]string{rawArchivePath}, archiveFile); err != nil {
		return nil, errors.Wrap(err, "failed to create archive")
	}
	b, err := ioutil.ReadFile(archiveFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive file")
	}

	return b, nil
}

// generateExpectedFilesystem uses kots to simularae a pull from a local source
// and then creates a tar from what the output is. because of this, it's expected
// that kots is working as expected when creating a new test
func generateExpectedFilesystem(rawArchivePath string) ([]byte, error) {
	tmpRootDir, err := ioutil.TempDir("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tmpRootDir)

	pullOptions := kotspull.PullOptions{
		RootDir:             tmpRootDir,
		LocalPath:           rawArchivePath,
		ExcludeKotsKinds:    true,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
		Silent:              true,
	}

	_, err = kotspull.Pull("replicated://integration", pullOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pull")
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}

	archiveDir, err := ioutil.TempDir("", "kots")
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
	if err := tarGz.Archive(paths, archiveFile); err != nil {
		return nil, errors.Wrap(err, "failed to create archive")
	}
	b, err := ioutil.ReadFile(archiveFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive file")
	}

	return b, nil
}
