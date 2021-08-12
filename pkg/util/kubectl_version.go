package util

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/blang/semver"
	"github.com/pkg/errors"
)

var knownKubectlVersions = []semver.Version{
	semver.MustParse("1.21.2"),
	semver.MustParse("1.20.4"),
	semver.MustParse("1.19.3"),
	semver.MustParse("1.18.10"),
	semver.MustParse("1.17.13"),
	semver.MustParse("1.16.3"),
	semver.MustParse("1.14.9"),
}

// finds a known version that matches the provided range
func matchKnownVersion(userString string) string {
	parsedRange, err := semver.ParseRange(userString)
	if err != nil {
		log.Printf("unable to parse range %s: %s", userString, err)
		return ""
	}

	// loop through list of known versions and check for matches
	for _, knownVersion := range knownKubectlVersions {
		if parsedRange(knownVersion) {
			return "v" + knownVersion.String()
		}
	}

	// otherwise return the empty string
	return ""
}

func FindKubectlVersion(userString string) (string, error) {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return "", errors.Wrap(err, "failed to find kubectl")
	}

	// then maybe override it with custom kubectl version
	if userString != "" && userString != "latest" {
		// matchKnownVersion only returns a string on success
		actualVersion := matchKnownVersion(userString)
		if actualVersion == "" {
			log.Printf("unable to find kubectl version matching %s, using default of 'latest'", userString)
			return kubectl, nil
		}

		customKubectl, err := exec.LookPath(fmt.Sprintf("kubectl-%s", actualVersion))
		if err != nil {
			log.Printf("unable to find custom kubectl version %s in path: %s, using default of 'latest'", actualVersion, err.Error())
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
