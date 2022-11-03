package upload

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
)

func createUploadableArchive(rootPath string) (string, error) {
	if strings.HasSuffix(rootPath, string(os.PathSeparator)) {
		rootPath = strings.TrimSuffix(rootPath, string(os.PathSeparator))
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}

	paths := []string{
		path.Join(rootPath, "upstream"),
		path.Join(rootPath, "base"),
		path.Join(rootPath, "overlays"),
	}

	skippedFilesPath := path.Join(rootPath, "skippedFiles")
	if _, err := os.Stat(skippedFilesPath); err == nil {
		paths = append(paths, skippedFilesPath)
	}

	// the caller of this function is repsonsible for deleting this file
	tempDir, err := ioutil.TempDir("", "kots")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	if err := tarGz.Archive(paths, path.Join(tempDir, "kots-uploadable-archive.tar.gz")); err != nil {
		return "", errors.Wrap(err, "failed to create tar gz")
	}

	return path.Join(tempDir, "kots-uploadable-archive.tar.gz"), nil
}

func findUpdateCursor(rootPath string) (string, error) {
	installationFilePath := path.Join(rootPath, "upstream", "userdata", "installation.yaml")
	_, err := os.Stat(installationFilePath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", errors.Wrap(err, "failed to open file")
	}

	installation, err := kotsutil.LoadInstallationFromPath(installationFilePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to load installation from path")
	}

	return installation.Spec.UpdateCursor, nil
}

func findLicense(rootPath string) (*string, error) {
	licenseFilePath := path.Join(rootPath, "upstream", "userdata", "license.yaml")
	_, err := os.Stat(licenseFilePath)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to open file with license")
	}

	b, err := ioutil.ReadFile(licenseFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read license file")
	}

	license := string(b)
	return &license, nil
}
