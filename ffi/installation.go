package main

import "C"

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/mholt/archiver"
)

//export ReadInstallation
func ReadInstallation(socket, archivePath string) {
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

		tmpRoot, err := ioutil.TempDir("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp path: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}
		defer os.RemoveAll(tmpRoot)

		// extract the current archive to this root
		tarGz := archiver.TarGz{
			Tar: &archiver.Tar{
				ImplicitTopLevelFolder: false,
			},
		}
		if err := tarGz.Unarchive(archivePath, tmpRoot); err != nil {
			fmt.Printf("failed to unarchive: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		installationFilePath := path.Join(tmpRoot, "upstream", "userdata", "installation.yaml")
		_, err = os.Stat(installationFilePath)
		if os.IsNotExist(err) {
			fmt.Printf("not installation data: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}
		if err != nil {
			fmt.Printf("failed to find file: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		installationData, err := ioutil.ReadFile(installationFilePath)
		if err != nil {
			fmt.Printf("failed to read file: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		ffiResult = NewFFIResult(0).WithData(string(installationData))
	}()
}
