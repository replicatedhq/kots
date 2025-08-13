package util

import (
	"regexp"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
)

// Utility method to extract the kube version from an EC version.
// Given a version string like "2.4.0+k8s-1.30-rc0", it returns the kube semver version "1.30"
func extractKubeVersion(ecVersion string) (*semver.Version, error) {
	re := regexp.MustCompile(`\+k8s-(\d+\.\d+)`)
	matches := re.FindStringSubmatch(ecVersion)
	if len(matches) != 2 {
		return nil, errors.Errorf("failed to extract kube version from '%s'", ecVersion)
	}
	kubeVersion, err := semver.NewVersion(matches[1])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse kube version from '%s'", ecVersion)
	}
	return kubeVersion, nil
}

// UpdateWithinKubeRange checks if the update version is within the same major version and
// at most one minor version ahead of the current version. Returns error if the update is not valid.
func UpdateWithinKubeRange(currentVersion, updateVersion string) error {
	current, err := extractKubeVersion(currentVersion)
	if err != nil {
		return errors.Wrap(err, "failed to extract current kube version")
	}
	update, err := extractKubeVersion(updateVersion)
	if err != nil {
		return errors.Wrap(err, "failed to extract update kube version")
	}
	if current.Major() != update.Major() {
		return errors.Errorf("major version mismatch: current %s, update %s", current, update)
	}
	if current.GreaterThan(update) {
		return errors.Errorf("cannot downgrade the kubernetes version: current %s, update %s", current, update)
	}
	if update.Minor() > current.Minor()+1 {
		return errors.Errorf("cannot update by more than one kubernetes minor version: current %s, update %s", current, update)
	}
	return nil
}
