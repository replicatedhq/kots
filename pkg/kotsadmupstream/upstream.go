package upstream

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	identity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
)

func DownloadUpdate(appID string, update types.Update, skipPreflights bool, skipCompatibilityCheck bool) (sequence int64, finalError error) {
	if err := store.GetStore().SetTaskStatus("update-download", "Fetching update...", "running"); err != nil {
		return 0, errors.Wrap(err, "failed to set task status")
	}

	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := store.GetStore().UpdateTaskStatusTimestamp("update-download"); err != nil {
					logger.Error(err)
				}
			case <-finishedCh:
				return
			}
		}
	}()

	defer func() {
		if finalError == nil {
			if err := store.GetStore().ClearTaskStatus("update-download"); err != nil {
				logger.Error(err)
			}
		} else {
			errMsg := finalError.Error()
			if cause, ok := errors.Cause(finalError).(util.ActionableError); ok {
				errMsg = cause.Error()
			}
			if err := store.GetStore().SetTaskStatus("update-download", errMsg, "failed"); err != nil {
				logger.Error(err)
			}
		}
	}()

	archiveDir, baseSequence, err := store.GetStore().GetAppVersionBaseArchive(appID, update.VersionLabel)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get base archive dir for version %s", update.VersionLabel)
	}
	defer os.RemoveAll(archiveDir)

	beforeKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return 0, errors.Wrap(err, "failed to read kots kinds before update")
	}

	beforeCursor := beforeKotsKinds.Installation.Spec.UpdateCursor

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := store.GetStore().SetTaskStatus("update-download", scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get app")
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to list downstreams for app")
	}

	downstreamNames := []string{}
	for _, d := range downstreams {
		downstreamNames = append(downstreamNames, d.Name)
	}

	appNamespace := util.AppNamespace()

	appSequence, err := store.GetStore().GetNextAppSequence(a.ID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get new app sequence")
	}

	latestLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get latest license")
	}

	identityConfigFile := filepath.Join(archiveDir, "upstream", "userdata", "identityconfig.yaml")
	if _, err := os.Stat(identityConfigFile); os.IsNotExist(err) {
		file, err := identity.InitAppIdentityConfig(a.Slug, kotsv1beta1.Storage{})
		if err != nil {
			return 0, errors.Wrap(err, "failed to init identity config")
		}
		identityConfigFile = file
		defer os.Remove(identityConfigFile)
	} else if err != nil {
		return 0, errors.Wrap(err, "failed to get stat identity config file")
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(appID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get registry settings")
	}

	pullOptions := kotspull.PullOptions{
		LicenseObj:          latestLicense,
		Namespace:           appNamespace,
		ConfigFile:          filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"),
		IdentityConfigFile:  identityConfigFile,
		InstallationFile:    filepath.Join(archiveDir, "upstream", "userdata", "installation.yaml"),
		UpdateCursor:        update.Cursor,
		RootDir:             archiveDir,
		Downstreams:         downstreamNames,
		ExcludeKotsKinds:    true,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
		ReportWriter:        pipeWriter,
		AppSlug:             a.Slug,
		AppSequence:         appSequence,
		IsGitOps:            a.IsGitOps,
		ReportingInfo:       reporting.GetReportingInfo(a.ID),
		RewriteImages:       registrySettings.IsValid(),
		RewriteImageOptions: kotspull.RewriteImageOptions{
			Host:       registrySettings.Hostname,
			Namespace:  registrySettings.Namespace,
			Username:   registrySettings.Username,
			Password:   registrySettings.Password,
			IsReadOnly: registrySettings.IsReadOnly,
		},
		SkipCompatibilityCheck: skipCompatibilityCheck,
	}

	if _, err := kotspull.Pull(fmt.Sprintf("replicated://%s", beforeKotsKinds.License.Spec.AppSlug), pullOptions); err != nil {
		return 0, errors.Wrap(err, "failed to pull")
	}

	afterKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return 0, errors.Wrap(err, "failed to read kots kinds after update")
	}

	if afterKotsKinds.Installation.Spec.UpdateCursor == beforeCursor {
		return 0, nil // ?
	}

	newSequence, err := store.GetStore().CreateAppVersion(a.ID, &baseSequence, archiveDir, "Upstream Update", skipPreflights, &version.DownstreamGitOps{})
	if err != nil {
		return 0, errors.Wrap(err, "failed to create version")
	}

	if !skipPreflights {
		if err := preflight.Run(appID, a.Slug, newSequence, a.IsAirgap, archiveDir); err != nil {
			return 0, errors.Wrap(err, "failed to run preflights")
		}
	}

	return newSequence, nil
}
