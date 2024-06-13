package upgradeservice

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	identity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/pull"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/tasks"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/replicatedhq/kots/pkg/util"
)

func bootstrap(params types.UpgradeServiceParams) error {
	// TODO NOW: airgap mode

	if err := pullArchiveFromOnline(params); err != nil {
		return errors.Wrap(err, "failed to pull archive from online")
	}

	return nil
}

func pullArchiveFromOnline(params types.UpgradeServiceParams) (finalError error) {
	license, err := kotsutil.LoadLicenseFromBytes([]byte(params.AppLicense))
	if err != nil {
		return errors.Wrap(err, "failed to load license from bytes")
	}

	beforeKotsKinds, err := kotsutil.LoadKotsKinds(params.BaseArchive)
	if err != nil {
		return errors.Wrap(err, "failed to load current kotskinds")
	}

	if err := pull.CleanBaseArchive(params.BaseArchive); err != nil {
		return errors.Wrap(err, "failed to clean base archive")
	}

	registrySettings := registrytypes.RegistrySettings{
		Hostname:   params.RegistryEndpoint,
		Username:   params.RegistryUsername,
		Password:   params.RegistryPassword,
		Namespace:  params.RegistryNamespace,
		IsReadOnly: params.RegistryIsReadOnly,
	}

	identityConfigFile := filepath.Join(params.BaseArchive, "upstream", "userdata", "identityconfig.yaml")
	if _, err := os.Stat(identityConfigFile); os.IsNotExist(err) {
		file, err := identity.InitAppIdentityConfig(params.AppSlug)
		if err != nil {
			return errors.Wrap(err, "failed to init identity config")
		}
		identityConfigFile = file
		defer os.Remove(identityConfigFile)
	} else if err != nil {
		return errors.Wrap(err, "failed to get stat identity config file")
	}

	pipeReader, pipeWriter := io.Pipe()
	defer func() {
		pipeWriter.CloseWithError(finalError)
	}()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := tasks.SetTaskStatus("update-download", scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	pullOptions := pull.PullOptions{
		LicenseObj:          license,
		Namespace:           util.AppNamespace(),
		ConfigFile:          filepath.Join(params.BaseArchive, "upstream", "userdata", "config.yaml"),
		IdentityConfigFile:  identityConfigFile,
		InstallationFile:    filepath.Join(params.BaseArchive, "upstream", "userdata", "installation.yaml"),
		UpdateCursor:        params.UpdateCursor,
		RootDir:             params.BaseArchive,
		Downstreams:         []string{"this-cluster"},
		ExcludeKotsKinds:    true,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
		ReportWriter:        pipeWriter,
		AppID:               params.AppID,
		AppSlug:             params.AppSlug,
		AppSequence:         params.NextSequence,
		IsGitOps:            params.AppIsGitOps,
		ReportingInfo:       params.ReportingInfo,
		RewriteImages:       registrySettings.IsValid(),
		RewriteImageOptions: registrySettings,
		KotsKinds:           beforeKotsKinds,
	}

	_, err = pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions)
	if err != nil {
		return errors.Wrap(err, "failed to pull")
	}

	return nil
}
