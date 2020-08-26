package online

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"
	kotsadmconfig "github.com/replicatedhq/kots/kotsadm/pkg/config"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/online/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/updatechecker"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	"github.com/replicatedhq/kots/pkg/pull"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func CreateAppFromOnline(pendingApp *types.PendingApp, upstreamURI string, isAutomated bool) (_ *kotsutil.KotsKinds, finalError error) {
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

	configValues, err := readConfigValuesFromInClusterSecret()
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
		ReportWriter:        pipeWriter,
		AppSlug:             pendingApp.Slug,
		AppSequence:         0,
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

	newSequence, err := version.CreateFirstVersion(pendingApp.ID, tmpRoot, "Online Install")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new version")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(tmpRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kotskinds from path")
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

		needsConfig, err := kotsadmconfig.NeedsConfiguration(configSpec, configValuesSpec, licenseSpec)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if app needs configuration")
		}

		if !needsConfig {
			err := downstream.SetDownstreamVersionPendingPreflight(pendingApp.ID, newSequence)
			if err != nil {
				return nil, errors.Wrap(err, "failed to set downstream version status to 'pending preflight'")
			}
		}
	}

	if err := preflight.Run(pendingApp.ID, newSequence, tmpRoot); err != nil {
		return nil, errors.Wrap(err, "failed to start preflights")
	}

	return kotsKinds, nil
}

func readConfigValuesFromInClusterSecret() (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to create clientset")
	}

	configValuesSecrets, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "kots.io/automation=configvalues",
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to list configvalues secrets")
	}

	// just get the first
	for _, configValuesSecret := range configValuesSecrets.Items {
		configValues, ok := configValuesSecret.Data["configvalues"]
		if !ok {
			logger.Errorf("config values secret %q does not contain config values key", configValuesSecret.Name)
			continue
		}

		// delete it, these are one time use secrets
		err = clientset.CoreV1().Secrets(configValuesSecret.Namespace).Delete(context.TODO(), configValuesSecret.Name, metav1.DeleteOptions{})
		if err != nil {
			logger.Errorf("error deleting config values secret: %v", err)
		}

		return string(configValues), nil
	}

	return "", nil
}
