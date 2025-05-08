package upgradeservice

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	identity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/pull"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	upgradepreflight "github.com/replicatedhq/kots/pkg/upgradeservice/preflight"
	"github.com/replicatedhq/kots/pkg/upgradeservice/task"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/replicatedhq/kots/pkg/util"
)

func bootstrap(params types.UpgradeServiceParams) (finalError error) {
	if err := updateWithinKubeRange(params); err != nil {
		return errors.Wrap(err, "kube version update is not within allowed range. Please update at most one minor version at a time")
	}

	if err := k8sutil.InitHelmCapabilities(); err != nil {
		return errors.Wrap(err, "failed to init helm capabilities")
	}
	if err := upgradepreflight.Init(); err != nil {
		return errors.Wrap(err, "failed to init preflight")
	}
	if params.AppIsAirgap {
		if err := pullArchiveFromAirgap(params); err != nil {
			return errors.Wrap(err, "failed to pull archive from airgap")
		}
	} else {
		if err := pullArchiveFromOnline(params); err != nil {
			return errors.Wrap(err, "failed to pull archive from online")
		}
	}
	return nil
}

// updateWithinKubeRange checks if the update version is within the same major version and
// at most one minor version ahead of the current version.
func updateWithinKubeRange(params types.UpgradeServiceParams) error {
	currentVersion, err := extractKubeVersion(params.CurrentECVersion)
	if err != nil {
		return errors.Wrap(err, "failed to extract current kube version")
	}
	updateVersion, err := extractKubeVersion(params.UpdateECVersion)
	if err != nil {
		return errors.Wrap(err, "failed to extract update kube version")
	}
	if currentVersion.Major() != updateVersion.Major() {
		return errors.Errorf("major version mismatch: current %s, update %s", currentVersion, updateVersion)
	}
	if updateVersion.Minor() > currentVersion.Minor()+1 {
		return errors.Errorf("cannot update more than one minor version: current %s, update %s", currentVersion, updateVersion)
	}
	return nil
}

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

func pullArchiveFromAirgap(params types.UpgradeServiceParams) (finalError error) {
	airgapRoot, err := archives.ExtractAppMetaFromAirgapBundle(params.UpdateAirgapBundle)
	if err != nil {
		return errors.Wrap(err, "failed to extract archive")
	}
	defer os.RemoveAll(airgapRoot)

	pullOptions := pull.PullOptions{
		IsAirgap:     true,
		AirgapRoot:   airgapRoot,
		AirgapBundle: params.UpdateAirgapBundle,
		Silent:       true,
	}
	if err := pullArchive(params, pullOptions); err != nil {
		return errors.Wrap(err, "failed to pull")
	}
	return nil
}

func pullArchiveFromOnline(params types.UpgradeServiceParams) (finalError error) {
	pullOptions := pull.PullOptions{
		IsGitOps:      params.AppIsGitOps,
		ReportingInfo: params.ReportingInfo,
	}
	if err := pullArchive(params, pullOptions); err != nil {
		return errors.Wrap(err, "failed to pull")
	}
	return nil
}

func pullArchive(params types.UpgradeServiceParams, pullOptions pull.PullOptions) (finalError error) {
	license, err := kotsutil.LoadLicenseFromBytes([]byte(params.AppLicense))
	if err != nil {
		return errors.Wrap(err, "failed to load license from bytes")
	}

	// In the upgrade service, it may be the case that the environment variables do not exist in
	// the container, as we are running in a previous release of the helm chart. If this is the
	// case, we fall back to the previous behavior and get the endpoint from the license.
	if val := os.Getenv("REPLICATED_APP_ENDPOINT"); val == "" {
		os.Setenv("REPLICATED_APP_ENDPOINT", license.Spec.Endpoint)
	}

	identityConfigFile, err := getIdentityConfigFile(params)
	if err != nil {
		return errors.Wrap(err, "failed to get identity config file")
	}

	beforeKotsKinds, err := kotsutil.LoadKotsKinds(params.AppArchive)
	if err != nil {
		return errors.Wrap(err, "failed to load current kotskinds")
	}

	if err := pull.CleanBaseArchive(params.AppArchive); err != nil {
		return errors.Wrap(err, "failed to clean base archive")
	}

	registrySettings := registrytypes.RegistrySettings{
		Hostname:   params.RegistryEndpoint,
		Username:   params.RegistryUsername,
		Password:   params.RegistryPassword,
		Namespace:  params.RegistryNamespace,
		IsReadOnly: params.RegistryIsReadOnly,
	}

	pipeReader, pipeWriter := io.Pipe()
	defer func() {
		pipeWriter.CloseWithError(finalError)
	}()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := task.SetStatusStarting(params.AppSlug, scanner.Text()); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	// common options
	pullOptions.LicenseObj = license
	pullOptions.Namespace = util.AppNamespace()
	pullOptions.ConfigFile = filepath.Join(params.AppArchive, "upstream", "userdata", "config.yaml")
	pullOptions.InstallationFile = filepath.Join(params.AppArchive, "upstream", "userdata", "installation.yaml")
	pullOptions.IdentityConfigFile = identityConfigFile
	pullOptions.UpdateCursor = params.UpdateCursor
	pullOptions.RootDir = params.AppArchive
	pullOptions.Downstreams = []string{"this-cluster"}
	pullOptions.ExcludeKotsKinds = true
	pullOptions.ExcludeAdminConsole = true
	pullOptions.CreateAppDir = false
	pullOptions.ReportWriter = pipeWriter
	pullOptions.AppID = params.AppID
	pullOptions.AppSlug = params.AppSlug
	pullOptions.AppSequence = params.NextSequence
	pullOptions.RewriteImages = registrySettings.IsValid()
	pullOptions.RewriteImageOptions = registrySettings
	pullOptions.KotsKinds = beforeKotsKinds

	_, err = pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions)
	if err != nil && errors.Cause(err) != pull.ErrConfigNeeded {
		return errors.Wrap(err, "failed to pull")
	}

	return nil
}

func getIdentityConfigFile(params types.UpgradeServiceParams) (string, error) {
	identityConfigFile := filepath.Join(params.AppArchive, "upstream", "userdata", "identityconfig.yaml")
	if _, err := os.Stat(identityConfigFile); os.IsNotExist(err) {
		file, err := identity.InitAppIdentityConfig(params.AppSlug)
		if err != nil {
			return "", errors.Wrap(err, "failed to init identity config")
		}
		identityConfigFile = file
		defer os.Remove(identityConfigFile)
	} else if err != nil {
		return "", errors.Wrap(err, "failed to get stat identity config file")
	}
	return identityConfigFile, nil
}
