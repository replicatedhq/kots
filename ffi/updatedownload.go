package main

import "C"

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/replicatedhq/kots/pkg/pull"
)

//export UpdateDownload
func UpdateDownload(socket, fromArchivePath, namespace, registryJson, cursor string) {
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

		registryInfo := struct {
			Host      string `json:"registryHostname"`
			Username  string `json:"registryUsername"`
			Password  string `json:"registryPassword"`
			Namespace string `json:"namespace"`
		}{}
		if err := json.Unmarshal([]byte(registryJson), &registryInfo); err != nil {
			fmt.Printf("failed to unmarshal registry info: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		tmpRoot, err := ioutil.TempDir("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp path: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}
		defer os.RemoveAll(tmpRoot)

		tarGz, err := extractArchive(tmpRoot, fromArchivePath)
		if err != nil {
			fmt.Printf("failed to extract archive: %s\n", err.Error())
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
		license, err := loadLicenseFromPath(expectedLicenseFile)
		if err != nil {
			fmt.Printf("failed to load license: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		pullOptions := pull.PullOptions{
			LicenseFile:         expectedLicenseFile,
			Namespace:           namespace,
			ConfigFile:          filepath.Join(tmpRoot, "upstream", "userdata", "config.yaml"),
			UpdateCursor:        cursor,
			RootDir:             tmpRoot,
			ExcludeKotsKinds:    true,
			ExcludeAdminConsole: true,
			CreateAppDir:        false,
			ReportWriter:        statusClient.getOutputWriter(),
		}

		if registryInfo.Host != "" {
			pullOptions.RewriteImages = true
			pullOptions.RewriteImageOptions = pull.RewriteImageOptions{
				Host:      registryInfo.Host,
				Namespace: registryInfo.Namespace,
				Username:  registryInfo.Username,
				Password:  registryInfo.Password,
			}
		}

		if _, err := pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions); err != nil {
			fmt.Printf("failed to pull upstream: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		afterCursor, err := readCursorFromPath(installationFilePath)
		if err != nil {
			fmt.Printf("failed to read cursor file after update: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		fmt.Printf("Result of checking for updates for %s: Before: %s, After %s\n", license.Spec.AppSlug, beforeCursor, afterCursor)

		isUpdateAvailable := string(beforeCursor) != string(afterCursor)
		if !isUpdateAvailable {
			ffiResult = NewFFIResult(0)
			return
		}

		paths := []string{
			filepath.Join(tmpRoot, "upstream"),
			filepath.Join(tmpRoot, "base"),
			filepath.Join(tmpRoot, "overlays"),
		}

		err = os.Remove(fromArchivePath)
		if err != nil {
			fmt.Printf("failed to delete archive to replace: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		if err := tarGz.Archive(paths, fromArchivePath); err != nil {
			fmt.Printf("failed to write archive: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		ffiResult = NewFFIResult(1)
	}()
}
