package main

import "C"

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mholt/archiver"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/rewrite"
	"k8s.io/client-go/kubernetes/scheme"
)

//export PullFromLicense
func PullFromLicense(socket string, licenseData string, downstream string, outputFile string) {
	go func() {
		var ffiResult *FFIResult

		statusClient, err := connectToStatusServer(socket)
		if err != nil {
			fmt.Printf("failed to connect to status server: %s\n", err)
			return
		}
		defer func() {
			statusClient.end(ffiResult)
		}()

		license, err := loadLicense(licenseData)
		if err != nil {
			fmt.Printf("failed to load license: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		licenseFile, err := ioutil.TempFile("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp file: %s\n", err)
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		defer os.Remove(licenseFile.Name())

		if err := ioutil.WriteFile(licenseFile.Name(), []byte(licenseData), 0644); err != nil {
			fmt.Printf("failed to write license to temp file: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		// pull to a tmp dir
		tmpRoot, err := ioutil.TempDir("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp root path: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
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
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		// make an archive
		tarGz := archiver.TarGz{
			Tar: &archiver.Tar{
				ImplicitTopLevelFolder: true,
			},
		}

		paths := []string{
			filepath.Join(tmpRoot, "upstream"),
			filepath.Join(tmpRoot, "base"),
			filepath.Join(tmpRoot, "overlays"),
		}

		if err := tarGz.Archive(paths, outputFile); err != nil {
			fmt.Printf("failed to write archive: %s", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		ffiResult = NewFFIResult(0)
	}()
}

//export RewriteImagesInVersion
func RewriteImagesInVersion(socket, fromArchivePath, outputFile, downstreamsStr, k8sNamespace, registry, username, password, namespace string) {
	go func() {
		var ffiResult *FFIResult

		statusClient, err := connectToStatusServer(socket)
		if err != nil {
			fmt.Printf("failed to connect to status server: %s\n", err)
			return
		}
		defer func() {
			statusClient.end(ffiResult)
		}()

		donwstreams := []string{}
		err = json.Unmarshal([]byte(downstreamsStr), &donwstreams)
		if err != nil {
			if err != nil {
				fmt.Printf("failed to decode downstreams: %s\n", err.Error())
				ffiResult = NewFFIResult(1).WithError(err)
				return
			}
		}

		tmpRoot, err := ioutil.TempDir("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp root path: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		defer os.RemoveAll(tmpRoot)

		tarGz := archiver.TarGz{
			Tar: &archiver.Tar{
				ImplicitTopLevelFolder: false,
			},
		}
		if err := tarGz.Unarchive(fromArchivePath, tmpRoot); err != nil {
			fmt.Printf("failed to unarchive: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		installationFilePath := filepath.Join(tmpRoot, "upstream", "userdata", "installation.yaml")
		beforeCursor, err := readCursorFromPath(installationFilePath)
		if err != nil {
			fmt.Printf("failed to read cursor file: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		expectedLicenseFile := filepath.Join(tmpRoot, "upstream", "userdata", "license.yaml")
		_, err = os.Stat(expectedLicenseFile)
		if err != nil {
			fmt.Printf("failed to find license file in archive\n")
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}
		licenseData, err := ioutil.ReadFile(expectedLicenseFile)
		if err != nil {
			fmt.Printf("failed to read license file: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(licenseData), nil, nil)
		if err != nil {
			fmt.Printf("failed to decode license data: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}
		license := obj.(*kotsv1beta1.License)

		configValues, err := parseConfigValuesFromFile(filepath.Join(tmpRoot, "upstream", "userdata", "config.yaml"))
		if err != nil {
			fmt.Printf("failed to decode config values: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		options := rewrite.RewriteOptions{
			RootDir:           tmpRoot,
			UpstreamURI:       fmt.Sprintf("replicated://%s", license.Spec.AppSlug),
			UpstreamPath:      filepath.Join(tmpRoot, "upstream"),
			LocalCursor:       beforeCursor,
			Downstreams:       donwstreams,
			Silent:            true,
			CreateAppDir:      false,
			ExcludeKotsKinds:  true,
			License:           license,
			ConfigValues:      configValues,
			K8sNamespace:      k8sNamespace,
			ReportWriter:      statusClient.getOutputWriter(),
			RegistryEndpoint:  registry,
			RegistryUsername:  username,
			RegistryPassword:  password,
			RegistryNamespace: namespace,
		}

		if err := rewrite.Rewrite(options); err != nil {
			fmt.Printf("failed to pull upstream: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		paths := []string{
			filepath.Join(tmpRoot, "upstream"),
			filepath.Join(tmpRoot, "base"),
			filepath.Join(tmpRoot, "overlays"),
		}

		if err := tarGz.Archive(paths, outputFile); err != nil {
			fmt.Printf("failed to write archive: %s", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		ffiResult = NewFFIResult(0)
	}()
}
