package main

import "C"

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/upstream"
)

//export ListUpdates
func ListUpdates(socket, fromArchivePath, currentCursor string) {
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

		if _, err := extractArchive(tmpRoot, fromArchivePath); err != nil {
			fmt.Printf("failed to extract archive: %s\n", err.Error())
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

		peekOptions := pull.PeekOptions{
			LicenseFile:   expectedLicenseFile,
			CurrentCursor: currentCursor,
		}

		updates, err := pull.Peek(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), peekOptions)
		if err != nil {
			fmt.Printf("failed to peek upstream: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}
		if updates == nil {
			updates = []upstream.Update{}
		}

		b, err := json.Marshal(updates)
		if err != nil {
			fmt.Printf("failed to marshal updates: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}
		ffiResult = NewFFIResult(0).WithData(string(b))
	}()
}
