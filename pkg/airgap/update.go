package airgap

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/cursor"
	identity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/tasks"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func UpdateAppFromAirgap(a *apptypes.App, airgapBundlePath string, deploy bool, skipPreflights bool, skipCompatibilityCheck bool) (finalError error) {
	finishedChan := make(chan error)
	defer close(finishedChan)

	tasks.StartUpdateTaskMonitor("update-download", finishedChan)
	defer func() {
		finishedChan <- finalError
	}()

	if err := store.GetStore().SetTaskStatus("update-download", "Extracting files...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	airgapRoot, err := extractAppMetaFromAirgapBundle(airgapBundlePath)
	if err != nil {
		return errors.Wrap(err, "failed to extract archive")
	}
	defer os.RemoveAll(airgapRoot)

	err = UpdateAppFromPath(a, airgapRoot, airgapBundlePath, deploy, skipPreflights, skipCompatibilityCheck)
	if err != nil {
		return errors.Wrap(err, "failed to update app")
	}

	return nil
}

func UpdateAppFromPath(a *apptypes.App, airgapRoot string, airgapBundlePath string, deploy bool, skipPreflights bool, skipCompatibilityCheck bool) error {
	if err := store.GetStore().SetTaskStatus("update-download", "Processing package...", "running"); err != nil {
		return errors.Wrap(err, "failed to set tasks status")
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get app registry settings")
	}

	airgap, err := kotsutil.FindAirgapMetaInDir(airgapRoot)
	if err != nil {
		return errors.Wrap(err, "failed to find airgap meta")
	}

	missingPrereqs, err := GetMissingRequiredVersions(a, airgap)
	if err != nil {
		return errors.Wrapf(err, "failed to check required versions")
	}

	if len(missingPrereqs) > 0 {
		return util.ActionableError{
			NoRetry: true,
			Message: fmt.Sprintf("This airgap bundle cannot be uploaded because versions %s are required and must be uploaded first.", strings.Join(missingPrereqs, ", ")),
		}
	}

	archiveDir, baseSequence, err := store.GetStore().GetAppVersionBaseArchive(a.ID, airgap.Spec.VersionLabel)
	if err != nil {
		return errors.Wrapf(err, "failed to get base archive dir for version %s", airgap.Spec.VersionLabel)
	}
	defer os.RemoveAll(archiveDir)

	beforeKotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to load current kotskinds")
	}

	if beforeKotsKinds.License == nil {
		err := errors.New("no license found in application")
		return err
	}

	if err := store.GetStore().SetTaskStatus("update-download", "Processing app package...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	appNamespace := util.AppNamespace()

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to list downstreams for app")
	}

	downstreamNames := []string{}
	for _, d := range downstreams {
		downstreamNames = append(downstreamNames, d.Name)
	}

	if err := store.GetStore().SetTaskStatus("update-download", "Creating app version...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	appSequence, err := store.GetStore().GetNextAppSequence(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get new app sequence")
	}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := store.GetStore().SetTaskStatus("update-download", scanner.Text(), "running"); err != nil {
				logger.Error(errors.Wrap(err, "failed to update download status"))
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	// Using license from db instead of upstream bundle because the one in db has not been re-marshalled
	license, err := kotsutil.LoadLicenseFromBytes([]byte(a.License))
	if err != nil {
		return errors.Wrap(err, "failed parse license")
	}

	identityConfigFile := filepath.Join(archiveDir, "upstream", "userdata", "identityconfig.yaml")
	if _, err := os.Stat(identityConfigFile); os.IsNotExist(err) {
		file, err := identity.InitAppIdentityConfig(a.Slug)
		if err != nil {
			return errors.Wrap(err, "failed to init identity config")
		}
		identityConfigFile = file
		defer os.Remove(identityConfigFile)
	} else if err != nil {
		return errors.Wrap(err, "failed to get stat identity config file")
	}

	if err := pull.CleanBaseArchive(archiveDir); err != nil {
		return errors.Wrap(err, "failed to clean base archive")
	}

	pullOptions := pull.PullOptions{
		Downstreams:            downstreamNames,
		LicenseObj:             license,
		Namespace:              appNamespace,
		ConfigFile:             filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"),
		IdentityConfigFile:     identityConfigFile,
		AirgapRoot:             airgapRoot,
		AirgapBundle:           airgapBundlePath,
		InstallationFile:       filepath.Join(archiveDir, "upstream", "userdata", "installation.yaml"),
		UpdateCursor:           beforeKotsKinds.Installation.Spec.UpdateCursor,
		RootDir:                archiveDir,
		ExcludeKotsKinds:       true,
		ExcludeAdminConsole:    true,
		CreateAppDir:           false,
		ReportWriter:           pipeWriter,
		Silent:                 true,
		RewriteImages:          true,
		RewriteImageOptions:    registrySettings,
		AppID:                  a.ID,
		AppSlug:                a.Slug,
		AppSequence:            appSequence,
		SkipCompatibilityCheck: skipCompatibilityCheck,
		KotsKinds:              beforeKotsKinds,
	}

	if _, err := pull.Pull(fmt.Sprintf("replicated://%s", beforeKotsKinds.License.Spec.AppSlug), pullOptions); err != nil {
		if errors.Cause(err) != pull.ErrConfigNeeded {
			return errors.Wrap(err, "failed to pull")
		}
	}

	afterKotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to read after kotskinds")
	}

	if err := canInstall(beforeKotsKinds, afterKotsKinds); err != nil {
		return errors.Wrap(err, "cannot install")
	}

	// Create the app in the db
	newSequence, err := store.GetStore().CreateAppVersion(a.ID, &baseSequence, archiveDir, "Airgap Update", skipPreflights, &version.DownstreamGitOps{}, render.Renderer{})
	if err != nil {
		return errors.Wrap(err, "failed to create new version")
	}

	hasStrictPreflights, err := store.GetStore().HasStrictPreflights(a.ID, newSequence)
	if err != nil {
		return errors.Wrap(err, "failed to check if app preflight has strict analyzers")
	}

	if hasStrictPreflights && skipPreflights {
		logger.Warnf("preflights will not be skipped, strict preflights are set to %t", hasStrictPreflights)
	}

	if !skipPreflights || hasStrictPreflights {
		if err := preflight.Run(a.ID, a.Slug, newSequence, true, archiveDir); err != nil {
			return errors.Wrap(err, "failed to start preflights")
		}
	}

	if deploy {
		downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
		if err != nil {
			return errors.Wrap(err, "failed to fetch downstreams")
		}
		if len(downstreams) == 0 {
			return errors.Errorf("no downstreams found for app %q", a.Slug)
		}
		downstream := downstreams[0]

		status, err := store.GetStore().GetStatusForVersion(a.ID, downstream.ClusterID, newSequence)
		if err != nil {
			return errors.Wrapf(err, "failed to get status for version %d", newSequence)
		}

		if status == storetypes.VersionPendingConfig {
			return errors.Errorf("not deploying version %d because it's %s", newSequence, status)
		}

		if err := version.DeployVersion(a.ID, newSequence); err != nil {
			return errors.Wrap(err, "failed to deploy app version")
		}
	}

	return nil
}

func canInstall(beforeKotsKinds *kotsutil.KotsKinds, afterKotsKinds *kotsutil.KotsKinds) error {
	var beforeSemver, afterSemver *semver.Version
	if v, err := semver.ParseTolerant(beforeKotsKinds.Installation.Spec.VersionLabel); err == nil {
		beforeSemver = &v
	}
	if v, err := semver.ParseTolerant(afterKotsKinds.Installation.Spec.VersionLabel); err == nil {
		afterSemver = &v
	}

	isSameVersion := false

	if beforeSemver != nil && afterSemver != nil {
		// Allow uploading older versions if both have semvers because they can be sorted correctly.
		if beforeSemver.EQ(*afterSemver) {
			isSameVersion = true
		}
	} else if beforeSemver != nil {
		// TODO: pass or fail?
	} else if afterSemver != nil {
		// TODO: pass or fail?
	} else {
		bc, err := cursor.NewCursor(beforeKotsKinds.Installation.Spec.UpdateCursor)
		if err != nil {
			return errors.Wrap(err, "failed to create bc")
		}

		ac, err := cursor.NewCursor(afterKotsKinds.Installation.Spec.UpdateCursor)
		if err != nil {
			return errors.Wrap(err, "failed to create ac")
		}

		if !bc.Comparable(ac) {
			return errors.Errorf("cannot compare %q and %q", beforeKotsKinds.Installation.Spec.UpdateCursor, afterKotsKinds.Installation.Spec.UpdateCursor)
		}

		installChannelID := beforeKotsKinds.Installation.Spec.ChannelID
		licenseChannelID := beforeKotsKinds.License.Spec.ChannelID
		installChannelName := beforeKotsKinds.Installation.Spec.ChannelName
		licenseChannelName := beforeKotsKinds.License.Spec.ChannelName
		if (installChannelID != "" && licenseChannelID != "" && installChannelID == licenseChannelID) || (installChannelName == licenseChannelName) {
			if bc.Equal(ac) {
				isSameVersion = true
			}
		}
	}

	if isSameVersion {
		return util.ActionableError{
			NoRetry: true,
			Message: fmt.Sprintf("Version %s (%s) cannot be installed again because it is already the current version", afterKotsKinds.Installation.Spec.VersionLabel, afterKotsKinds.Installation.Spec.UpdateCursor),
		}
	}

	return nil
}

func GetMissingRequiredVersions(app *apptypes.App, airgap *kotsv1beta1.Airgap) ([]string, error) {
	appVersions, err := store.GetStore().FindDownstreamVersions(app.ID, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get downstream versions")
	}

	license, err := kotsutil.LoadLicenseFromBytes([]byte(app.License))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load license")
	}

	return getMissingRequiredVersions(airgap, license, appVersions.AllVersions)
}

func getMissingRequiredVersions(airgap *kotsv1beta1.Airgap, license *kotsv1beta1.License, installedVersions []*downstreamtypes.DownstreamVersion) ([]string, error) {
	missingVersions := make([]string, 0)
	if len(installedVersions) == 0 {
		return missingVersions, nil
	}

	for _, requiredRelease := range airgap.Spec.RequiredReleases {
		laterReleaseInstalled := false
		for _, appVersion := range installedVersions {
			requiredSemver, requiredSemverErr := semver.ParseTolerant(requiredRelease.VersionLabel)

			// semvers can be compared across channels
			// if a semmver is missing, fallback to comparing the cursor but only if channel is the same
			if license.Spec.IsSemverRequired && appVersion.Semver != nil && requiredSemverErr == nil {
				if requiredSemver.LE(*appVersion.Semver) {
					laterReleaseInstalled = true
					break
				}
			} else {
				// cursors can only be compared on the same channel
				if appVersion.ChannelID != airgap.Spec.ChannelID {
					continue
				}
				if appVersion.Cursor == nil {
					return nil, errors.Errorf("cursor required but version %s does not have cursor", appVersion.UpdateCursor)
				}
				requiredCursor, err := cursor.NewCursor(requiredRelease.UpdateCursor)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse required update cursor %q", requiredRelease.UpdateCursor)
				}
				if requiredCursor.Before(*appVersion.Cursor) || requiredCursor.Equal(*appVersion.Cursor) {
					laterReleaseInstalled = true
					break
				}
			}
		}

		if !laterReleaseInstalled {
			missingVersions = append([]string{requiredRelease.VersionLabel}, missingVersions...)
		} else {
			break
		}
	}

	return missingVersions, nil
}
