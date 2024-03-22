package airgap

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/airgap/types"
	"github.com/replicatedhq/kots/pkg/archives"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmconfig "github.com/replicatedhq/kots/pkg/kotsadmconfig"
	identity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

type CreateAirgapAppOpts struct {
	PendingApp             *types.PendingApp
	AirgapPath             string
	RegistryHost           string
	RegistryNamespace      string
	RegistryUsername       string
	RegistryPassword       string
	RegistryIsReadOnly     bool
	IsAutomated            bool
	SkipPreflights         bool
	SkipCompatibilityCheck bool
}

// CreateAppFromAirgap does a lot. Maybe too much. Definitely too much.
// This function assumes that there's an app in the database that doesn't have a version
// After execution, there will be a sequence 0 of the app, and all clusters in the database
// will also have a version
func CreateAppFromAirgap(opts CreateAirgapAppOpts) (finalError error) {
	taskID := fmt.Sprintf("airgap-install-slug-%s", opts.PendingApp.Slug)
	if err := store.GetStore().SetTaskStatus(taskID, "Processing package...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := store.GetStore().UpdateTaskStatusTimestamp(taskID); err != nil {
					logger.Error(errors.Wrapf(err, "failed to update task %s", taskID))
				}
			case <-finishedCh:
				return
			}
		}
	}()

	defer func() {
		if finalError == nil {
			if err := store.GetStore().ClearTaskStatus(taskID); err != nil {
				logger.Error(errors.Wrap(err, "failed to clear install task status"))
			}
			if err := store.GetStore().SetAppInstallState(opts.PendingApp.ID, "installed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set app status to installed"))
			}
		} else {
			if err := store.GetStore().SetTaskStatus(taskID, finalError.Error(), "failed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set error on install task status"))
			}
			if err := store.GetStore().SetAppInstallState(opts.PendingApp.ID, "airgap_upload_error"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set app status to error"))
			}
		}
	}()

	if err := store.GetStore().SetAppIsAirgap(opts.PendingApp.ID, true); err != nil {
		return errors.Wrap(err, "failed to set app is airgap")
	}

	// Extract it
	if err := store.GetStore().SetTaskStatus(taskID, "Extracting files...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	airgapBundle := ""
	archiveDir := opts.AirgapPath
	if strings.ToLower(filepath.Ext(opts.AirgapPath)) == ".airgap" {
		// on the api side, headless intalls don't have the airgap file
		dir, err := extractAppMetaFromAirgapBundle(opts.AirgapPath)
		if err != nil {
			return errors.Wrap(err, "failed to extract archive")
		}
		defer os.RemoveAll(dir)

		airgapBundle = opts.AirgapPath
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

	if err := store.GetStore().SetTaskStatus(taskID, "Reading license data...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(opts.PendingApp.LicenseData), nil, nil)
	if err != nil {
		return errors.Wrap(err, "failed to read pending license data")
	}
	license := obj.(*kotsv1beta1.License)

	licenseFile, err := ioutil.TempFile("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	if err := ioutil.WriteFile(licenseFile.Name(), []byte(opts.PendingApp.LicenseData), 0644); err != nil {
		os.Remove(licenseFile.Name())
		return errors.Wrapf(err, "failed to write license to temp file")
	}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := store.GetStore().SetTaskStatus(taskID, scanner.Text(), "running"); err != nil {
				logger.Error(errors.Wrapf(err, "failed to set status for task %s", taskID))
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	appNamespace := util.AppNamespace()

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

	identityConfigFile, err := identity.InitAppIdentityConfig(opts.PendingApp.Slug)
	if err != nil {
		return errors.Wrap(err, "failed to init identity config")
	}
	defer os.Remove(identityConfigFile)

	if opts.RegistryPassword == registrytypes.PasswordMask {
		// On initial install, registry info can be copied from kotsadm config,
		// and password in this case will not be included in the request.
		kotsadmSettings, err := registry.GetKotsadmRegistry()
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to load kotsadm config"))
		} else if kotsadmSettings.Hostname == opts.RegistryHost {
			opts.RegistryPassword = kotsadmSettings.Password
		}
	}

	instParams, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		return errors.Wrap(err, "failed to get existing kotsadm config map")
	}

	pullOptions := pull.PullOptions{
		Downstreams:         []string{"this-cluster"},
		LocalPath:           releaseDir,
		Namespace:           appNamespace,
		LicenseFile:         licenseFile.Name(),
		ConfigFile:          configFile,
		IdentityConfigFile:  identityConfigFile,
		IsAirgap:            true,
		AirgapRoot:          archiveDir,
		AirgapBundle:        airgapBundle,
		Silent:              !opts.IsAutomated,
		ExcludeKotsKinds:    true,
		RootDir:             tmpRoot,
		ExcludeAdminConsole: true,
		RewriteImages:       true,
		ReportWriter:        pipeWriter,
		RewriteImageOptions: registrytypes.RegistrySettings{
			Hostname:   opts.RegistryHost,
			Namespace:  opts.RegistryNamespace,
			Username:   opts.RegistryUsername,
			Password:   opts.RegistryPassword,
			IsReadOnly: opts.RegistryIsReadOnly,
		},
		AppID:                  opts.PendingApp.ID,
		AppSlug:                opts.PendingApp.Slug,
		AppSequence:            0,
		AppVersionLabel:        instParams.AppVersionLabel,
		SkipCompatibilityCheck: opts.SkipCompatibilityCheck,
	}

	if _, err := pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions); err != nil {
		if errors.Cause(err) != pull.ErrConfigNeeded {
			return errors.Wrap(err, "failed to pull")
		}
	}

	if err := store.GetStore().AddAppToAllDownstreams(opts.PendingApp.ID); err != nil {
		return errors.Wrap(err, "failed to add app to all downstreams")
	}

	a, err := store.GetStore().GetApp(opts.PendingApp.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get app from pending app")
	}

	if err := store.GetStore().UpdateRegistry(opts.PendingApp.ID, opts.RegistryHost, opts.RegistryUsername, opts.RegistryPassword, opts.RegistryNamespace, opts.RegistryIsReadOnly); err != nil {
		return errors.Wrap(err, "failed to update registry")
	}

	// yes, again in case of errors
	if err := store.GetStore().SetAppIsAirgap(opts.PendingApp.ID, true); err != nil {
		return errors.Wrap(err, "failed to set app is airgap the second time")
	}

	newSequence, err := store.GetStore().CreateAppVersion(a.ID, nil, tmpRoot, "Airgap Install", opts.SkipPreflights, &version.DownstreamGitOps{}, render.Renderer{})
	if err != nil {
		return errors.Wrap(err, "failed to create new version")
	}

	troubleshootOpts := supportbundletypes.TroubleshootOptions{
		InCluster: true,
	}
	_, err = supportbundle.CreateSupportBundleDependencies(a, newSequence, troubleshootOpts)
	if err != nil {
		return errors.Wrap(err, "failed to create support bundle dependencies")
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(tmpRoot)
	if err != nil {
		return errors.Wrap(err, "failed to load kotskinds from path")
	}

	status, err := store.GetStore().GetDownstreamVersionStatus(opts.PendingApp.ID, newSequence)
	if err != nil {
		return errors.Wrap(err, "failed to get downstream version status")
	}

	if status == storetypes.VersionPendingClusterManagement {
		// if pending cluster management, we don't want to deploy the app
		return nil
	}

	hasStrictPreflights, err := store.GetStore().HasStrictPreflights(a.ID, newSequence)
	if err != nil {
		return errors.Wrap(err, "failed to check if app preflight has strict analyzers")
	}

	if hasStrictPreflights && opts.SkipPreflights {
		logger.Warnf("preflights will not be skipped, strict preflights are set to %t", hasStrictPreflights)
	}

	if opts.IsAutomated && kotsKinds.IsConfigurable() {
		// bypass the config screen if no configuration is required
		registrySettings := registrytypes.RegistrySettings{
			Hostname:   opts.RegistryHost,
			Namespace:  opts.RegistryNamespace,
			Username:   opts.RegistryUsername,
			Password:   opts.RegistryPassword,
			IsReadOnly: opts.RegistryIsReadOnly,
		}
		needsConfig, err := kotsadmconfig.NeedsConfiguration(a.Slug, newSequence, a.IsAirgap, kotsKinds, registrySettings)
		if err != nil {
			return errors.Wrap(err, "failed to check if app needs configuration")
		}
		if !needsConfig {
			if err := store.GetStore().SetDownstreamVersionStatus(opts.PendingApp.ID, newSequence, storetypes.VersionPending, ""); err != nil {
				return errors.Wrap(err, "failed to set downstream version status to pending")
			}
			if opts.SkipPreflights && !hasStrictPreflights {
				if err := version.DeployVersion(opts.PendingApp.ID, newSequence); err != nil {
					return errors.Wrap(err, "failed to deploy version")
				}
			} else {
				err := store.GetStore().SetDownstreamVersionStatus(opts.PendingApp.ID, newSequence, storetypes.VersionPendingPreflight, "")
				if err != nil {
					return errors.Wrap(err, "failed to set downstream version status to 'pending preflight'")
				}
			}
		}
	}

	if !opts.SkipPreflights || hasStrictPreflights {
		if err := preflight.Run(opts.PendingApp.ID, opts.PendingApp.Slug, newSequence, true, tmpRoot); err != nil {
			return errors.Wrap(err, "failed to start preflights")
		}
	}

	if !kotsKinds.IsConfigurable() && opts.SkipPreflights && !hasStrictPreflights {
		// app is not configurable and preflights are skipped, so just deploy the app
		if err := store.GetStore().SetDownstreamVersionStatus(opts.PendingApp.ID, newSequence, storetypes.VersionPending, ""); err != nil {
			return errors.Wrap(err, "failed to set downstream version status to pending")
		}
		if err := version.DeployVersion(opts.PendingApp.ID, newSequence); err != nil {
			return errors.Wrap(err, "failed to deploy version")
		}
	}

	err = kotsutil.RemoveAppVersionLabelFromInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to delete app version label from config"))
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

func extractAppMetaFromAirgapBundle(airgapBundle string) (string, error) {
	destDir, err := ioutil.TempDir("", "kotsadm-airgap-meta-")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	fileReader, err := os.Open(airgapBundle)
	if err != nil {
		return "", errors.Wrap(err, "failed to open file")
	}
	defer fileReader.Close()

	gzipReader, err := gzip.NewReader(fileReader)
	if err != nil {
		return "", errors.Wrap(err, "failed to get new gzip reader")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", errors.Wrap(err, "failed to get read archive")
		}

		// First items in airgap archive are metadata files.
		// As soon as we see the first directory, we are hitting images.
		if header.Name == "." {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			break
		}

		err = func() error {
			fileName := filepath.Join(destDir, header.Name)

			fileWriter, err := os.Create(fileName)
			if err != nil {
				return errors.Wrapf(err, "failed to create file %q", header.Name)
			}

			defer fileWriter.Close()

			_, err = io.Copy(fileWriter, tarReader)
			if err != nil {
				return errors.Wrapf(err, "failed to write file %q", header.Name)
			}

			return nil
		}()
		if err != nil {
			return "", err
		}
	}

	return destDir, nil
}
