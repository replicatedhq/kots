package k8sutil

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chartutil"
)

func InitHelmCapabilities() error {
	clientset, err := GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	serverGroups, err := clientset.Discovery().ServerGroups()
	if err != nil {
		return errors.Wrap(err, "failed to get server groups")
	}

	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return errors.Wrap(err, "failed to get server version")
	}

	versionSet := chartutil.VersionSet{}
	for _, serverGroup := range serverGroups.Groups {
		for _, version := range serverGroup.Versions {
			versionSet = append(versionSet, version.GroupVersion)
		}
	}

	// Need to remove non-digits from minor version to make valid semver
	reg, err := regexp.Compile("[^0-9]+")
	if err != nil {
		return errors.Wrap(err, "failed to compile regex")
	}

	chartutil.DefaultCapabilities = &chartutil.Capabilities{
		KubeVersion: chartutil.KubeVersion{
			Version: fmt.Sprintf("v%s.%s.0", serverVersion.Major, reg.ReplaceAllString(serverVersion.Minor, "")),
			Major:   serverVersion.Major,
			Minor:   serverVersion.Minor,
		},
		APIVersions: versionSet,
		HelmVersion: chartutil.DefaultCapabilities.HelmVersion,
	}

	return nil
}
