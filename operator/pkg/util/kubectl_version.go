package util

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

var knownKubectlVersions = []string{
	"v1.14.7",
	"v1.16.1",
}

// finds a known version that 'matches' the provided userstring
// for instance, 14.x matches v1.14.7, and 1.16 matches v1.16.1
func matchKnownVersion(userString string) string {
	// strip trailing 'x'
	userString = strings.TrimSuffix(userString, "x")
	userString = strings.TrimSuffix(userString, "X")

	// loop through list of known versions and check for matches
	for _, knownVersion := range knownKubectlVersions {
		if strings.Contains(knownVersion, userString) {
			return knownVersion
		}
	}

	// otherwise return the raw string
	return userString
}

func FindKubectlVersion(userString string) (string, error) {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return "", errors.Wrap(err, "failed to find kubectl")
	}

	// then maybe override it with custom kubectl version
	if userString != "" && userString != "latest" {
		actualVersion := matchKnownVersion(userString)

		customKubectl, err := exec.LookPath(fmt.Sprintf("kubectl-%s", actualVersion))
		if err != nil {
			log.Printf("unable to find custom kubectl version %s in path: %s", actualVersion, err.Error())
		} else if customKubectl == "" {
			log.Printf("unable to find custom kubectl version %s in path, found empty string", actualVersion)
		} else {
			log.Printf("using custom kubectl version %s at %s", actualVersion, customKubectl)
			kubectl = customKubectl
		}
	} else if userString == "latest" {
		log.Printf("using latest kubectl version")
	} else {
		log.Printf("no kubectl version set, using default of 'latest'")
	}

	return kubectl, nil
}
