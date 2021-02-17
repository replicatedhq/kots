package airgap

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/airgap/types"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/crypto"
	kotsadmconfig "github.com/replicatedhq/kots/pkg/kotsadmconfig"
	downstream "github.com/replicatedhq/kots/pkg/kotsadmdownstream"
	identity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	"github.com/replicatedhq/kots/pkg/version"
	"k8s.io/client-go/kubernetes/scheme"
)

// CreateAppFromAirgap does a lot. Maybe too much. Definitely too much.
// This function assumes that there's an app in the database that doesn't have a version
// After execution, there will be a sequence 0 of the app, and all clusters in the database
// will also have a version
func CreateAppFromAirgap(pendingApp *types.PendingApp, airgapPath string, registryHost string, namespace string, username string, password string, isAutomated bool, skipPreflights bool) (finalError error) {
	if err := store.GetStore().SetTaskStatus("airgap-install", "Processing package...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := store.GetStore().UpdateTaskStatusTimestamp("airgap-install"); err != nil {
					logger.Error(err)
				}
			case <-finishedCh:
				return
			}
		}
	}()

	defer func() {
		if finalError == nil {
			if err := store.GetStore().ClearTaskStatus("airgap-install"); err != nil {
				logger.Error(errors.Wrap(err, "failed to clear install task status"))
			}
			if err := store.GetStore().SetAppInstallState(pendingApp.ID, "installed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set app status to installed"))
			}
		} else {
			if err := store.GetStore().SetTaskStatus("airgap-install", finalError.Error(), "failed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set error on install task status"))
			}
			if err := store.GetStore().SetAppInstallState(pendingApp.ID, "airgap_upload_error"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set app status to error"))
			}
		}
	}()

	if err := store.GetStore().SetAppIsAirgap(pendingApp.ID, true); err != nil {
		return errors.Wrap(err, "failed to set app is airgap")
	}

	// Extract it
	if err := store.GetStore().SetTaskStatus("airgap-install", "Extracting files...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	archiveDir := airgapPath
	if strings.ToLower(filepath.Ext(airgapPath)) == ".airgap" {
		// on the api side, headless intalls don't have the airgap file
		dir, err := version.ExtractArchiveToTempDirectory(airgapPath)
		if err != nil {
			return errors.Wrap(err, "failed to extract archive")
		}
		defer os.RemoveAll(dir)

		archiveDir = dir
	}

	// extract the release
	workspace, err := ioutil.TempDir("", "kots-airgap")
	if err != nil {
		return errors.Wrap(err, "failed to create workspace")
	}
	defer os.RemoveAll(workspace)

	releaseDir, err := extractAppRelease(workspace, archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to extract app dir")
	}

	tmpRoot, err := ioutil.TempDir("", "kots")
	if err != nil {
		return errors.Wrap(err, "failed to create temp root")
	}
	defer os.RemoveAll(tmpRoot)

	if err := store.GetStore().SetTaskStatus("airgap-install", "Reading license data...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(pendingApp.LicenseData), nil, nil)
	if err != nil {
		return errors.Wrap(err, "failed to read pending license data")
	}
	license := obj.(*kotsv1beta1.License)

	licenseFile, err := ioutil.TempFile("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	if err := ioutil.WriteFile(licenseFile.Name(), []byte(pendingApp.LicenseData), 0644); err != nil {
		os.Remove(licenseFile.Name())
		return errors.Wrapf(err, "failed to write license to temp file")
	}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := store.GetStore().SetTaskStatus("airgap-install", scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	configValues, err := kotsadmconfig.ReadConfigValuesFromInClusterSecret()
	if err != nil {
		return errors.Wrap(err, "failed to read config values from in cluster")
	}
	configFile := ""
	if configValues != "" {
		tmpFile, err := ioutil.TempFile("", "kots")
		if err != nil {
			return errors.Wrap(err, "failed to create temp file for config values")
		}
		defer os.RemoveAll(tmpFile.Name())
		if err := ioutil.WriteFile(tmpFile.Name(), []byte(configValues), 0644); err != nil {
			return errors.Wrap(err, "failed to write config values to temp file")
		}

		configFile = tmpFile.Name()
	}

	identityConfigFile, err := identity.InitAppIdentityConfig(pendingApp.Slug, kotsv1beta1.Storage{}, crypto.AESCipher{})
	if err != nil {
		return errors.Wrap(err, "failed to init identity config")
	}
	defer os.Remove(identityConfigFile)

	pullOptions := pull.PullOptions{
		Downstreams:         []string{"this-cluster"},
		LocalPath:           releaseDir,
		Namespace:           appNamespace,
		LicenseFile:         licenseFile.Name(),
		ConfigFile:          configFile,
		IdentityConfigFile:  identityConfigFile,
		AirgapRoot:          archiveDir,
		Silent:              true,
		ExcludeKotsKinds:    true,
		RootDir:             tmpRoot,
		ExcludeAdminConsole: true,
		RewriteImages:       true,
		ReportWriter:        pipeWriter,
		RewriteImageOptions: pull.RewriteImageOptions{
			ImageFiles: filepath.Join(archiveDir, "images"),
			Host:       registryHost,
			Namespace:  namespace,
			Username:   username,
			Password:   password,
		},
		AppSlug:     pendingApp.Slug,
		AppSequence: 0,
	}

	if _, err := pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions); err != nil {
		return errors.Wrap(err, "failed to pull")
	}

	if err := store.GetStore().AddAppToAllDownstreams(pendingApp.ID); err != nil {
		return errors.Wrap(err, "failed to add app to all downstreams")
	}

	a, err := store.GetStore().GetApp(pendingApp.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get app from pending app")
	}

	if password == registrytypes.PasswordMask {
		// On initial install, registry info can be copied from kotsadm config,
		// and password in this case will not be included in the request.
		kotsadmSettings, err := registry.GetKotsadmRegistry()
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to load kotsadm config"))
		} else if kotsadmSettings.Hostname == registryHost {
			password = kotsadmSettings.Password
		}
	}

	if err := store.GetStore().UpdateRegistry(pendingApp.ID, registryHost, username, password, namespace); err != nil {
		return errors.Wrap(err, "failed to update registry")
	}

	// yes, again in case of errors
	if err := store.GetStore().SetAppIsAirgap(pendingApp.ID, true); err != nil {
		return errors.Wrap(err, "failed to set app is airgap the second time")
	}

	newSequence, err := store.GetStore().CreateAppVersion(a.ID, nil, tmpRoot, "Airgap Upload", skipPreflights, &version.DownstreamGitOps{})
	if err != nil {
		return errors.Wrap(err, "failed to create new version")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(tmpRoot)
	if err != nil {
		return errors.Wrap(err, "failed to load kotskinds from path")
	}

	err = supportbundle.CreateRenderedSpec(a.ID, a.CurrentSequence, "", true, kotsKinds)
	if err != nil {
		return errors.Wrap(err, "failed to create rendered support bundle spec")
	}

	if isAutomated && kotsKinds.Config != nil {
		// bypass the config screen if no configuration is required
		licenseSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "License")
		if err != nil {
			return errors.Wrap(err, "failed to marshal license spec")
		}

		configSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Config")
		if err != nil {
			return errors.Wrap(err, "failed to marshal config spec")
		}

		configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
		if err != nil {
			return errors.Wrap(err, "failed to marshal configvalues spec")
		}

		identityConfigSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "IdentityConfig")
		if err != nil {
			return errors.Wrap(err, "failed to marshal identityconfig spec")
		}

		configOpts := kotsadmconfig.ConfigOptions{
			ConfigSpec:         configSpec,
			ConfigValuesSpec:   configValuesSpec,
			LicenseSpec:        licenseSpec,
			IdentityConfigSpec: identityConfigSpec,
			RegistryHost:       registryHost,
			RegistryNamespace:  namespace,
			RegistryUser:       username,
			RegistryPassword:   password,
		}

		needsConfig, err := kotsadmconfig.NeedsConfiguration(configOpts)
		if err != nil {
			return errors.Wrap(err, "failed to check if app needs configuration")
		}

		if !needsConfig {
			if skipPreflights {
				if err := version.DeployVersion(pendingApp.ID, newSequence); err != nil {
					return errors.Wrap(err, "failed to deploy version")
				}
			} else {
				err := downstream.SetDownstreamVersionPendingPreflight(pendingApp.ID, newSequence)
				if err != nil {
					return errors.Wrap(err, "failed to set downstream version status to 'pending preflight'")
				}
			}
		}
	}

	if !skipPreflights {
		if err := preflight.Run(pendingApp.ID, pendingApp.Slug, newSequence, true, tmpRoot); err != nil {
			return errors.Wrap(err, "failed to start preflights")
		}
	}

	return nil
}

func extractAppRelease(workspace string, airgapDir string) (string, error) {
	files, err := ioutil.ReadDir(airgapDir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read airgap dir")
	}

	destDir := filepath.Join(workspace, "extracted-app-release")
	if err := os.Mkdir(destDir, 0744); err != nil {
		return "", errors.Wrap(err, "failed to create tmp dir")
	}

	numExtracted := 0
	for _, file := range files {
		if file.IsDir() { // TODO: support nested dirs?
			continue
		}
		err := archives.ExtractTGZArchiveFromFile(filepath.Join(airgapDir, file.Name()), destDir)
		if err != nil {
			fmt.Printf("ignoring file %q: %v\n", file.Name(), err)
			continue
		}
		numExtracted++
	}

	if numExtracted == 0 {
		return "", errors.New("no release found in airgap archive")
	}

	return destDir, nil
}
