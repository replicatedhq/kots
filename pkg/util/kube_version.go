package util

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
)

var re = regexp.MustCompile(`\+k8s-(\d+\.\d+)`)

var (
	ErrInvalidKubeVersionFormat = errors.New("failed to extract kube version")
	ErrKubeMajorVersionUpgrade  = errors.New("major version mismatch")
	ErrKubeVersionDowngrade     = errors.New("cannot downgrade the kubernetes version")
	ErrKubeMinorRangeMismatch   = errors.New("cannot update by more than one kubernetes minor version")
)

// KubeMinorRangeError carries the versions needed to explain the upgrade path.
type KubeMinorRangeError struct {
	Current, Target string
	Updates         uint64
}

func (e *KubeMinorRangeError) Error() string {
	return fmt.Sprintf("This application version cannot be installed on this cluster yet.\n\n"+
		"The cluster is running Kubernetes %s, and this version includes Kubernetes %s. "+
		"Kubernetes can only be upgraded one minor version at a time, so this cluster needs at least %d updates to reach this version.\n\n"+
		"Contact your software vendor for the earlier application versions to install first.", e.Current, e.Target, e.Updates)
}

func (e *KubeMinorRangeError) Unwrap() error { return ErrKubeMinorRangeMismatch }

// Utility method to extract the kube version from an EC version.
// Given a version string like "2.4.0+k8s-1.30-rc0", it returns the kube semver version "1.30"
func extractKubeVersion(ecVersion string) (*semver.Version, error) {
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
		return fmt.Errorf("%w: %w", ErrInvalidKubeVersionFormat, err)
	}
	update, err := extractKubeVersion(updateVersion)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidKubeVersionFormat, err)
	}
	if current.Major() != update.Major() {
		return fmt.Errorf("%w: current %s, update %s", ErrKubeMajorVersionUpgrade, current, update)
	}
	if current.GreaterThan(update) {
		return fmt.Errorf("%w: current %s, update %s", ErrKubeVersionDowngrade, current, update)
	}
	if update.Minor() > current.Minor()+1 {
		return &KubeMinorRangeError{
			Current: fmt.Sprintf("%d.%d", current.Major(), current.Minor()),
			Target:  fmt.Sprintf("%d.%d", update.Major(), update.Minor()),
			Updates: update.Minor() - current.Minor(),
		}
	}
	return nil
}
