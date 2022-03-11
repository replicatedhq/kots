package online

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsadmconfig "github.com/replicatedhq/kots/pkg/kotsadmconfig"
	identity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/online/types"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
	"go.uber.org/zap"
)

type CreateOnlineAppOpts struct {
	PendingApp             *types.PendingApp
	UpstreamURI            string
	IsAutomated            bool
	SkipPreflights         bool
	SkipCompatibilityCheck bool
}

func CreateAppFromOnline(opts CreateOnlineAppOpts) (_ *kotsutil.KotsKinds, finalError error) {
	logger.Debug("creating app from online",
		zap.String("upstreamURI", opts.UpstreamURI))

	if err := store.GetStore().SetTaskStatus("online-install", "Uploading license...", "running"); err != nil {
		return nil, errors.Wrap(err, "failed to set task status")
	}

	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := store.GetStore().UpdateTaskStatusTimestamp("online-install"); err != nil {
					logger.Error(err)
				}
			case <-finishedCh:
				return
			}
		}
	}()

	defer func() {
		if finalError == nil {
			if err := store.GetStore().ClearTaskStatus("online-install"); err != nil {
				logger.Error(errors.Wrap(err, "failed to clear install task status"))
			}
			if err := store.GetStore().SetAppInstallState(opts.PendingApp.ID, "installed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set app status to installed"))
			}
			if err := updatechecker.Configure(opts.PendingApp.ID); err != nil {
				logger.Error(errors.Wrap(err, "failed to configure update checker"))
			}
		} else {
			if err := store.GetStore().SetTaskStatus("online-install", finalError.Error(), "failed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set error on install task status"))
			}
			if err := store.GetStore().SetAppInstallState(opts.PendingApp.ID, "install_error"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set app status to error"))
			}
		}
	}()

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := store.GetStore().SetTaskStatus("online-install", scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	// put the license in a temp file
	licenseFile, err := ioutil.TempFile("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create tmp file for license")
	}
	defer os.RemoveAll(licenseFile.Name())
	if err := ioutil.WriteFile(licenseFile.Name(), []byte(opts.PendingApp.LicenseData), 0644); err != nil {
		return nil, errors.Wrap(err, "failed to write license tmp file")
	}

	// pull to a tmp dir
	tmpRoot, err := ioutil.TempDir("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create tmp dir for pull")
	}
	defer os.RemoveAll(tmpRoot)

	appNamespace := util.AppNamespace()

	configValues, err := kotsadmconfig.ReadConfigValuesFromInClusterSecret()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config values from in cluster")
	}
	configFile := ""
	if configValues != "" {
		tmpFile, err := ioutil.TempFile("", "kots")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp file for config values")
		}
		defer os.RemoveAll(tmpFile.Name())
		if err := ioutil.WriteFile(tmpFile.Name(), []byte(configValues), 0644); err != nil {
			return nil, errors.Wrap(err, "failed to write config values to temp file")
		}

		configFile = tmpFile.Name()
	}

	identityConfigFile, err := identity.InitAppIdentityConfig(opts.PendingApp.Slug, kotsv1beta1.Storage{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to init identity config")
	}
	defer os.Remove(identityConfigFile)

	// kots install --config-values (and other documented automation workflows) support
	// a writing a config values file as a secret...
	// if this secret exists, we automatically (blindly) use it as the config values
	// for the application, and then delete it.
	pullOptions := pull.PullOptions{
		Downstreams:            []string{"this-cluster"},
		LicenseFile:            licenseFile.Name(),
		Namespace:              appNamespace,
		ExcludeKotsKinds:       true,
		RootDir:                tmpRoot,
		ExcludeAdminConsole:    true,
		CreateAppDir:           false,
		ConfigFile:             configFile,
		IdentityConfigFile:     identityConfigFile,
		ReportWriter:           pipeWriter,
		AppSlug:                opts.PendingApp.Slug,
		AppSequence:            0,
		ReportingInfo:          reporting.GetReportingInfo(opts.PendingApp.ID),
		SkipCompatibilityCheck: opts.SkipCompatibilityCheck,
	}

	if _, err := pull.Pull(opts.UpstreamURI, pullOptions); err != nil {
		return nil, errors.Wrap(err, "failed to pull")
	}

	// Create the downstream
	// copying this from typescript ...
	// i'll leave this next line
	// TODO: refactor this entire function to be testable, reliable and less monolithic
	if err := store.GetStore().AddAppToAllDownstreams(opts.PendingApp.ID); err != nil {
		return nil, errors.Wrap(err, "failed to add app to all downstreams")
	}
	if err := store.GetStore().SetAppIsAirgap(opts.PendingApp.ID, false); err != nil {
		return nil, errors.Wrap(err, "failed to set app is not airgap")
	}

	newSequence, err := store.GetStore().CreateAppVersion(opts.PendingApp.ID, nil, tmpRoot, "Online Install", opts.SkipPreflights, &version.DownstreamGitOps{}, render.Renderer{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new version")
	}

	troubleshootOpts := supportbundletypes.TroubleshootOptions{
		InCluster: true,
	}
	_, err = supportbundle.CreateSupportBundleDependencies(opts.PendingApp.ID, newSequence, troubleshootOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create rendered support bundle spec")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(tmpRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kotskinds from path")
	}

	hasStrictPreflights, err := store.GetStore().HasStrictPreflights(opts.PendingApp.ID, newSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if app preflight has strict analyzers")
	}

	if hasStrictPreflights && opts.SkipPreflights {
		logger.Warnf("preflights will not be skipped, strict preflights are set to %t", hasStrictPreflights)
	}

	if !opts.SkipPreflights || hasStrictPreflights {
		if err := preflight.Run(opts.PendingApp.ID, opts.PendingApp.Slug, newSequence, false, tmpRoot); err != nil {
			return nil, errors.Wrap(err, "failed to start preflights")
		}
	}

	if opts.IsAutomated && kotsKinds.IsConfigurable() {
		// bypass the config screen if no configuration is required and it's an automated install
		registrySettings, err := store.GetStore().GetRegistryDetailsForApp(opts.PendingApp.ID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get registry settings for app")
		}
		needsConfig, err := kotsadmconfig.NeedsConfiguration(opts.PendingApp.Slug, newSequence, false, kotsKinds, registrySettings)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if app needs configuration")
		}
		if !needsConfig {
			if opts.SkipPreflights || !hasStrictPreflights {
				if err := version.DeployVersion(opts.PendingApp.ID, newSequence); err != nil {
					return nil, errors.Wrap(err, "failed to deploy version")
				}
				go func() {
					if err := reporting.ReportAppInfo(opts.PendingApp.ID, newSequence, opts.SkipPreflights, opts.IsAutomated); err != nil {
						logger.Debugf("failed to send preflights data to replicated app: %v", err)
					}
				}()
			}
		}
	}

	if !kotsKinds.IsConfigurable() && opts.SkipPreflights && !hasStrictPreflights {
		// app is not configurable and preflights are skipped, so just deploy the app
		if err := version.DeployVersion(opts.PendingApp.ID, newSequence); err != nil {
			return nil, errors.Wrap(err, "failed to deploy version")
		}
		go func() {
			if err := reporting.ReportAppInfo(opts.PendingApp.ID, newSequence, opts.SkipPreflights, opts.IsAutomated); err != nil {
				logger.Debugf("failed to send preflights data to replicated app: %v", err)
			}
		}()
	}

	return kotsKinds, nil
}
