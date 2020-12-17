package online

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"
	kotsadmconfig "github.com/replicatedhq/kots/kotsadm/pkg/config"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/identity"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/online/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/reporting"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle"
	"github.com/replicatedhq/kots/kotsadm/pkg/updatechecker"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/pull"
	"go.uber.org/zap"
)

func CreateAppFromOnline(pendingApp *types.PendingApp, upstreamURI string, isAutomated bool, skipPreflights bool) (_ *kotsutil.KotsKinds, finalError error) {
	logger.Debug("creating app from online",
		zap.String("upstreamURI", upstreamURI))

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
			if err := store.GetStore().SetAppInstallState(pendingApp.ID, "installed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set app status to installed"))
			}
			if err := updatechecker.Configure(pendingApp.ID); err != nil {
				logger.Error(errors.Wrap(err, "failed to configure update checker"))
			}
		} else {
			if err := store.GetStore().SetTaskStatus("online-install", finalError.Error(), "failed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set error on install task status"))
			}
			if err := store.GetStore().SetAppInstallState(pendingApp.ID, "install_error"); err != nil {
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
	if err := ioutil.WriteFile(licenseFile.Name(), []byte(pendingApp.LicenseData), 0644); err != nil {
		return nil, errors.Wrap(err, "failed to write license tmp file")
	}

	// pull to a tmp dir
	tmpRoot, err := ioutil.TempDir("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create tmp dir for pull")
	}
	defer os.RemoveAll(tmpRoot)

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

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

	identityConfigFile, err := identity.InitAppIdentityConfig(pendingApp.Slug, kotsv1beta1.Storage{}, crypto.AESCipher{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to init identity config")
	}
	defer os.Remove(identityConfigFile)

	// kots install --config-values (and other documented automation workflows) support
	// a writing a config values file as a secret...
	// if this secret exists, we automatically (blindly) use it as the config values
	// for the application, and then delete it.
	pullOptions := pull.PullOptions{
		Downstreams:         []string{"this-cluster"},
		LicenseFile:         licenseFile.Name(),
		Namespace:           appNamespace,
		ExcludeKotsKinds:    true,
		RootDir:             tmpRoot,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
		ConfigFile:          configFile,
		IdentityConfigFile:  identityConfigFile,
		ReportWriter:        pipeWriter,
		AppSlug:             pendingApp.Slug,
		AppSequence:         0,
		ReportingInfo:       reporting.GetReportingInfo(pendingApp.ID),
	}

	if _, err := pull.Pull(upstreamURI, pullOptions); err != nil {
		return nil, errors.Wrap(err, "failed to pull")
	}

	// Create the downstream
	// copying this from typescript ...
	// i'll leave this next line
	// TODO: refactor this entire function to be testable, reliable and less monolithic
	if err := store.GetStore().AddAppToAllDownstreams(pendingApp.ID); err != nil {
		return nil, errors.Wrap(err, "failed to add app to all downstreams")
	}
	if err := store.GetStore().SetAppIsAirgap(pendingApp.ID, false); err != nil {
		return nil, errors.Wrap(err, "failed to set app is not airgap")
	}

	newSequence, err := version.CreateFirstVersion(pendingApp.ID, tmpRoot, "Online Install", skipPreflights)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new version")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(tmpRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kotskinds from path")
	}

	err = supportbundle.CreateRenderedSpec(pendingApp.ID, 0, "", true, kotsKinds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create rendered support bundle spec")
	}

	if isAutomated && kotsKinds.Config != nil {
		// bypass the config screen if no configuration is required
		licenseSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "License")
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal license spec")
		}

		configSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Config")
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal config spec")
		}

		configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal configvalues spec")
		}

		identityConfigSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "IdentityConfig")
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal identityconfig spec")
		}

		configOptions := kotsadmconfig.ConfigOptions{
			ConfigSpec:         configSpec,
			ConfigValuesSpec:   configValuesSpec,
			LicenseSpec:        licenseSpec,
			IdentityConfigSpec: identityConfigSpec,
			// TODO: are there ever registry settings here?
		}
		needsConfig, err := kotsadmconfig.NeedsConfiguration(configOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if app needs configuration")
		}

		if !needsConfig {
			if skipPreflights {
				if err := version.DeployVersion(pendingApp.ID, newSequence); err != nil {
					return nil, errors.Wrap(err, "failed to deploy version")
				}
			} else {
				err := downstream.SetDownstreamVersionPendingPreflight(pendingApp.ID, newSequence)
				if err != nil {
					return nil, errors.Wrap(err, "failed to set downstream version status to 'pending preflight'")
				}
			}
		}
	}

	if !skipPreflights {
		if err := preflight.Run(pendingApp.ID, pendingApp.Slug, newSequence, false, tmpRoot); err != nil {
			return nil, errors.Wrap(err, "failed to start preflights")
		}
	}

	return kotsKinds, nil
}
