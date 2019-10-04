package kotsadm

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/replicatedhq/kots/pkg/version"
)

var (
	OverrideVersion   = ""
	OverrideRegistry  = ""
	OverrideNamespace = ""
)

// return "alpha" for all prerelease or invalid versions of kots,
// kotsadm tag that matches this version for others
func kotsadmTag() string {
	if OverrideVersion != "" {
		return OverrideVersion
	}

	kotsVersion := version.Version()
	parsed, err := semver.NewVersion(kotsVersion)
	if err != nil {
		return "alpha"
	}

	if parsed.Prerelease() != "" {
		return "alpha"
	}

	return fmt.Sprintf("v%s", kotsVersion)
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
