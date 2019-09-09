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
