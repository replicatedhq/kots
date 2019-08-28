package main

import "C"

import (
	"os"
	"fmt"
	"path"
	"io/ioutil"

	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/mholt/archiver"

)


//export PullFromLicense
func PullFromLicense(licenseData string, downstream string, outputFile string) int { 
	licenseFile, err := ioutil.TempFile("", "kots")
	if err != nil {
		fmt.Printf("failed to create temp file: %#v\n", err)
		return 1
	}
	defer os.Remove(licenseFile.Name())

	if err := ioutil.WriteFile(licenseFile.Name(), []byte(licenseData), 0644); err != nil {
		fmt.Printf("failed to write license to temp file: %#v\n", err)
		return 1
	}

	// pull to a tmp dir
	tmpRoot, err := ioutil.TempDir("", "kots")
	if err != nil {
		fmt.Printf("failed to create temp root path: %#v\n", err)
		return 1
	}
	defer os.RemoveAll(tmpRoot)
	
	pullOptions := pull.PullOptions{
		Overwrite: true,
		Downstreams: []string{downstream},
		LicenseFile: licenseFile.Name(),
		ExcludeKotsKinds: true,
		RootDir: tmpRoot,
		ExcludeAdminConsole: true,
	}

	if err := pull.Pull("replicated://sentry-enterprise", pullOptions); err != nil {
		fmt.Printf("failed to pull upstream: %#v\n", err)
		return 1
	}

	// make an archive
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}

	// the tmp dir has the app name in it now
	subdirs, err := ioutil.ReadDir(tmpRoot)
	if err != nil {
		fmt.Printf("unable to read dir: %#v\n", err)
		return 1
	}

	uploadRootDir := ""
	for _, subdir := range subdirs {
		if subdir.IsDir() {
			if subdir.Name() == "." {
				continue
			}
			if subdir.Name() == ".." {
				continue
			}

			uploadRootDir = path.Join(tmpRoot, subdir.Name())
			break
		}
	}
	if uploadRootDir == "" {
		fmt.Println("failed to find upload root dir")
		return 1
	}

	paths := []string{
		path.Join(uploadRootDir, "upstream"),
		path.Join(uploadRootDir, "base"),
		path.Join(uploadRootDir, "overlays"),
	}

	if err := tarGz.Archive(paths, outputFile); err != nil {
		fmt.Printf("failed to write archive: %#v\n", err)
		return 1
	}

	return 0
}

func main() {}
