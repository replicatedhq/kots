package k8sutil

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	"helm.sh/helm/v4/pkg/chart/common"
)

func InitHelmCapabilities() error {
	clientset, err := GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	serverGroups, err := clientset.ServerGroups()
	if err != nil {
		return errors.Wrap(err, "failed to get server groups")
	}

	serverVersion, err := clientset.ServerVersion()
	if err != nil {
		return errors.Wrap(err, "failed to get server version")
	}

	versionSet := common.VersionSet{}
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

	kubeVersionStr := fmt.Sprintf("v%s.%s.0", serverVersion.Major, reg.ReplaceAllString(serverVersion.Minor, ""))
	kubeVersion, err := common.ParseKubeVersion(kubeVersionStr)
	if err != nil {
		return errors.Wrap(err, "failed to parse kubernetes version")
	}

	common.DefaultCapabilities = &common.Capabilities{
		KubeVersion: *kubeVersion,
		APIVersions: versionSet,
		HelmVersion: common.DefaultCapabilities.HelmVersion,
	}

	return nil
}
