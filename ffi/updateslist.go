package main

import "C"

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/upstream"
)

//export ListUpdates
func ListUpdates(socket, licenseData, currentCursor, currentChannel string) {
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

		licenseFile, err := writeLicenseFileFromLicenseData(licenseData)
		if err != nil {
			fmt.Printf("failed to write license file: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		defer os.Remove(licenseFile)

		getUpdatesOptions := pull.GetUpdatesOptions{
			LicenseFile:    licenseFile,
			CurrentCursor:  currentCursor,
			CurrentChannel: currentChannel,
			Silent:         true,
		}

		updates, err := pull.GetUpdates(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), getUpdatesOptions)
		if err != nil {
			fmt.Printf("failed to get updates for upstream: %s\n", err.Error())
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
