package plan

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/archives"
	identity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/update"
	"github.com/replicatedhq/kots/pkg/upgradeservice/task"
	"github.com/replicatedhq/kots/pkg/util"
)

func pullAppArchive(appSlug, versionLabel, updateCursor, channelID string) (appArchive string, baseSequence int64, finalError error) {
	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		return "", -1, errors.Wrap(err, "get app from slug")
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return "", -1, errors.Wrap(err, "get registry details for app")
	}

	baseArchive, baseSequence, err := store.GetStore().GetAppVersionBaseArchive(a.ID, versionLabel)
	if err != nil {
		return "", -1, errors.Wrap(err, "get app version base archive")
	}

	nextSequence, err := store.GetStore().GetNextAppSequence(a.ID)
	if err != nil {
		return "", -1, errors.Wrap(err, "get next app sequence")
	}

	license, err := kotsutil.LoadLicenseFromBytes([]byte(a.License))
	if err != nil {
		return "", -1, errors.Wrap(err, "parse app license")
	}

	airgapBundle := ""
	if a.IsAirgap {
		au, err := update.GetAirgapUpdate(a.Slug, channelID, updateCursor)
		if err != nil {
			return "", -1, errors.Wrap(err, "get airgap update")
		}
		airgapBundle = au
	}

	pullOptions := pull.PullOptions{}
	if a.IsAirgap {
		airgapRoot, err := archives.ExtractAppMetaFromAirgapBundle(airgapBundle)
		if err != nil {
			return "", -1, errors.Wrap(err, "extract archive")
		}
		defer os.RemoveAll(airgapRoot)

		pullOptions = pull.PullOptions{
			IsAirgap:     true,
			AirgapRoot:   airgapRoot,
			AirgapBundle: airgapBundle,
			Silent:       true,
		}
	} else {
		pullOptions = pull.PullOptions{
			IsGitOps: a.IsGitOps,
			// TODO (@salah)
			// ReportingInfo: params.ReportingInfo,
		}
	}

	identityConfigFile, err := getIdentityConfigFile(baseArchive, a.Slug)
	if err != nil {
		return "", -1, errors.Wrap(err, "get identity config file")
	}

	beforeKotsKinds, err := kotsutil.LoadKotsKinds(baseArchive)
	if err != nil {
		return "", -1, errors.Wrap(err, "load current kotskinds")
	}

	if err := pull.CleanBaseArchive(baseArchive); err != nil {
		return "", -1, errors.Wrap(err, "clean base archive")
	}

	pipeReader, pipeWriter := io.Pipe()
	defer func() {
		pipeWriter.CloseWithError(finalError)
	}()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := task.SetStatusStarting(a.Slug, scanner.Text()); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	// common options
	pullOptions.LicenseObj = license
	pullOptions.Namespace = util.AppNamespace()
	pullOptions.ConfigFile = filepath.Join(baseArchive, "upstream", "userdata", "config.yaml")
	pullOptions.InstallationFile = filepath.Join(baseArchive, "upstream", "userdata", "installation.yaml")
	pullOptions.IdentityConfigFile = identityConfigFile
	pullOptions.UpdateCursor = updateCursor
	pullOptions.RootDir = baseArchive
	pullOptions.Downstreams = []string{"this-cluster"}
	pullOptions.ExcludeKotsKinds = true
	pullOptions.ExcludeAdminConsole = true
	pullOptions.CreateAppDir = false
	pullOptions.ReportWriter = pipeWriter
	pullOptions.AppID = a.ID
	pullOptions.AppSlug = a.Slug
	pullOptions.AppSequence = nextSequence
	pullOptions.RewriteImages = registrySettings.IsValid()
	pullOptions.RewriteImageOptions = registrySettings
	pullOptions.KotsKinds = beforeKotsKinds

	_, err = pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions)
	if err != nil && errors.Cause(err) != pull.ErrConfigNeeded {
		return "", -1, errors.Wrap(err, "pull")
	}

	// base archive got updated during the pull process
	return baseArchive, baseSequence, nil
}

func getIdentityConfigFile(appArchive string, appSlug string) (string, error) {
	identityConfigFile := filepath.Join(appArchive, "upstream", "userdata", "identityconfig.yaml")
	if _, err := os.Stat(identityConfigFile); os.IsNotExist(err) {
		file, err := identity.InitAppIdentityConfig(appSlug)
		if err != nil {
			return "", errors.Wrap(err, "init identity config")
		}
		identityConfigFile = file
		defer os.Remove(identityConfigFile)
	} else if err != nil {
		return "", errors.Wrap(err, "get stat identity config file")
	}
	return identityConfigFile, nil
}
