package main

import "C"

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/mholt/archiver"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/pull"
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

		kotsscheme.AddToScheme(scheme.Scheme)
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(licenseData), nil, nil)
		if err != nil {
			fmt.Printf("failed to decode license data: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		license := obj.(*kotsv1beta1.License)

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
			path.Join(tmpRoot, "upstream"),
			path.Join(tmpRoot, "base"),
			path.Join(tmpRoot, "overlays"),
		}

		if err := tarGz.Archive(paths, outputFile); err != nil {
			fmt.Printf("failed to write archive: %s", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		ffiResult = NewFFIResult(0)
	}()
}
