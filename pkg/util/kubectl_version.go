package util

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
)

var knownKubectlVersions = []kubectlFuzzyVersion{
	{semver: semver.MustParse("1.21.1")},
	{semver: semver.MustParse("1.20.1")},
	{semver: semver.MustParse("1.19.1")},
	{semver: semver.MustParse("1.18.1")},
	{semver: semver.MustParse("1.17.1")},
	{semver: semver.MustParse("1.16.1")},
	{semver: semver.MustParse("1.14.1")},
}

type kubectlFuzzyVersion struct {
	semver semver.Version
}

func (v kubectlFuzzyVersion) String() string {
	return fmt.Sprintf("v%d.%d", v.semver.Major, v.semver.Minor)
}

func (v kubectlFuzzyVersion) Match(userString string) bool {
	if userString == "" || userString == "latest" {
		return false
	}

	if exactVer, err := semver.Parse(userString); err == nil {
		return exactVer.Major == v.semver.Major && exactVer.Minor == v.semver.Minor
	}

	rangeVer, err := semver.ParseRange(userString)
	if err != nil {
		return false
	}

	return rangeVer(v.semver)
}

// finds a known version that matches the provided range
func matchKnownVersion(userString string) string {
	// loop through list of known versions and check for matches
	for _, knownVersion := range knownKubectlVersions {
		if knownVersion.Match(userString) {
			return knownVersion.String()
		}
	}

	// otherwise return the empty string
	return ""
}

func FindKubectlVersion(userString string) (string, error) {
	if persistence.IsSQlite() {
		// in the kots run workflow, binaries exist under {kotsdatadir}/binaries
		return fmt.Sprintf("%s/binaries/kubectl", os.Getenv("KOTS_DATA_DIR")), nil
	}

	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return "", errors.Wrap(err, "failed to find kubectl")
	}

	// then maybe override it with custom kubectl version
	if userString == "latest" {
		log.Printf("using latest kubectl version")
	} else if userString != "" {
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
	} else {
		log.Printf("no kubectl version set, using default of 'latest'")
	}

	return kubectl, nil
}
