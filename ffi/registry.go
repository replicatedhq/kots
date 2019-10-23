package main

import "C"

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/docker/registry"
)

//export TestRegistryCredentials
func TestRegistryCredentials(socket, endpoint, username, password, org string) {
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

		err = registry.TestPushAccess(endpoint, username, password, org)
		if err != nil {
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		ffiResult = NewFFIResult(0)
	}()
}
