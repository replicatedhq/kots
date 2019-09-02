package main

import "C"

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/pull"
	"k8s.io/client-go/kubernetes/scheme"
)

//export UpdateCheck
func UpdateCheck(fromArchivePath string) int {
	tmpRoot, err := ioutil.TempDir("", "kots")
	if err != nil {
		fmt.Printf("failed to create temp path: %s\n", err.Error())
		return -1
	}
	defer os.RemoveAll(tmpRoot)

	// extract the current archive to this root
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(fromArchivePath, tmpRoot); err != nil {
		fmt.Printf("failed to unarchive: %s\n", err.Error())
		return -1
	}

	beforeCursor, err := readCursorFromPath(tmpRoot)
	if err != nil {
		fmt.Printf("failed to read cursor file: %s\n", err.Error())
		return -1
	}

	expectedLicenseFile := path.Join(tmpRoot, "upstream", "userdata", "license.yaml")
	_, err = os.Stat(expectedLicenseFile)
	if err != nil {
		fmt.Printf("failed to find license file in archive\n")
		return -1
	}
	licenseData, err := ioutil.ReadFile(expectedLicenseFile)
	if err != nil {
		fmt.Printf("failed to read license file: %s\n", err.Error())
		return -1
	}

	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(licenseData), nil, nil)
	if err != nil {
		fmt.Printf("failed to decode license data: %s\n", err.Error())
		return -1
	}
	license := obj.(*kotsv1beta1.License)

	pullOptions := pull.PullOptions{
		LicenseFile:         expectedLicenseFile,
		RootDir:             tmpRoot,
		ExcludeKotsKinds:    true,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
	}

	if _, err := pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions); err != nil {
		fmt.Printf("failed to pull upstream: %s\n", err.Error())
		return 1
	}

	afterCursor, err := readCursorFromPath(tmpRoot)
	if err != nil {
		fmt.Printf("failed to read cursor file after update: %s\n", err.Error())
		return -1
	}

	fmt.Printf("Result of checking for updates for %s: Before: %s, After %s\n", license.Spec.AppSlug, beforeCursor, afterCursor)

	isUpdateAvailable := string(beforeCursor) != string(afterCursor)
	if !isUpdateAvailable {
		return 0
	}

	paths := []string{
		path.Join(tmpRoot, "upstream"),
		path.Join(tmpRoot, "base"),
		path.Join(tmpRoot, "overlays"),
	}

	err = os.Remove(fromArchivePath)
	if err != nil {
		fmt.Printf("failed to delete archive to replace: %s\n", err.Error())
		return -1
	}

	if err := tarGz.Archive(paths, fromArchivePath); err != nil {
		fmt.Printf("failed to write archive: %s\n", err.Error())
		return -1
	}

	return 1
}

//export PullFromLicense
func PullFromLicense(licenseData string, downstream string, outputFile string) int {
	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(licenseData), nil, nil)
	if err != nil {
		fmt.Printf("failed to decode license data: %s\n", err.Error())
		return 1
	}
	license := obj.(*kotsv1beta1.License)

	licenseFile, err := ioutil.TempFile("", "kots")
	if err != nil {
		fmt.Printf("failed to create temp file: %s\n", err)
		return 1
	}
	defer os.Remove(licenseFile.Name())

	if err := ioutil.WriteFile(licenseFile.Name(), []byte(licenseData), 0644); err != nil {
		fmt.Printf("failed to write license to temp file: %s\n", err.Error())
		return 1
	}

	// pull to a tmp dir
	tmpRoot, err := ioutil.TempDir("", "kots")
	if err != nil {
		fmt.Printf("failed to create temp root path: %s\n", err.Error())
		return 1
	}
	defer os.RemoveAll(tmpRoot)

	pullOptions := pull.PullOptions{
		Downstreams:         []string{downstream},
		LicenseFile:         licenseFile.Name(),
		ExcludeKotsKinds:    true,
		RootDir:             tmpRoot,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
	}

	if _, err := pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions); err != nil {
		fmt.Printf("failed to pull upstream: %s\n", err.Error())
		return 1
	}

	// make an archive
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}

	paths := []string{
		path.Join(tmpRoot, "upstream"),
		path.Join(tmpRoot, "base"),
		path.Join(tmpRoot, "overlays"),
	}

	if err := tarGz.Archive(paths, outputFile); err != nil {
		fmt.Printf("failed to write archive: %s", err.Error())
		return 1
	}

	return 0
}

func readCursorFromPath(rootPath string) (string, error) {
	installationFilePath := path.Join(rootPath, "upstream", "userdata", "installation.yaml")
	_, err := os.Stat(installationFilePath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", errors.Wrap(err, "failed to open file")
	}

	installationData, err := ioutil.ReadFile(installationFilePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read update installation file")
	}

	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(installationData), nil, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to devode installation data")
	}

	installation := obj.(*kotsv1beta1.Installation)
	return installation.Spec.UpdateCursor, nil
}

func main() {}
