package kotsadm

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/replicatedhq/kots/pkg/version"
)

var (
	OverrideVersion   = ""
	OverrideRegistry  = ""
	OverrideNamespace = ""
)

// return "alpha" for all invalid versions of kots,
// kotsadm tag that matches this version for others
func kotsadmTag() string {
	if OverrideVersion != "" {
		return OverrideVersion
	}

	kotsVersion := version.Version()

	return kotsadmTagForVersionString(kotsVersion)
}

func kotsadmTagForVersionString(kotsVersion string) string {
	version, err := semver.NewVersion(kotsVersion)
	if err != nil {
		return "alpha"
	}

	if strings.Contains(version.Prerelease(), "dirty") {
		return "alpha"
	}

	if !strings.HasPrefix(kotsVersion, "v") {
		kotsVersion = fmt.Sprintf("v%s", kotsVersion)
	}

	return kotsVersion
}

func kotsadmRegistry() string {
	if OverrideRegistry == "" {
		if OverrideNamespace == "" {
			return "kotsadm"
		} else {
			return OverrideNamespace
		}
	}

	if OverrideNamespace == "" {
		return fmt.Sprintf("%s/kotsadm", OverrideRegistry)
	}

	return fmt.Sprintf("%s/%s", OverrideRegistry, OverrideNamespace)
}
