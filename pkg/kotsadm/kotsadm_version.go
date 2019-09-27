package kotsadm

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/replicatedhq/kots/pkg/version"
)

// return "alpha" for all prerelease or invalid versions of kots,
// kotsadm tag that matches this version for others
func kotsadmTag() string {
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
