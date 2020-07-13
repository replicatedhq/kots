package online

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/task"
	"github.com/replicatedhq/kots/kotsadm/pkg/updatechecker"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	kotsadmconfig "github.com/replicatedhq/kots/kotsadm/pkg/config"
	"github.com/replicatedhq/kots/pkg/pull"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type PendingApp struct {
	ID          string
	Slug        string
	Name        string
	LicenseData string
}

func CreateAppFromOnline(pendingApp *PendingApp, upstreamURI string, isAutomated bool) (_ *kotsutil.KotsKinds, finalError error) {
	logger.Debug("creating app from online",
		zap.String("upstreamURI", upstreamURI))

	if err := task.SetTaskStatus("online-install", "Uploading license...", "running"); err != nil {
		return nil, errors.Wrap(err, "failed to set task status")
	}

	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := task.UpdateTaskStatusTimestamp("online-install"); err != nil {
					logger.Error(err)
				}
			case <-finishedCh:
				return
			}
		}
	}()

	defer func() {
		if finalError == nil {
			if err := task.ClearTaskStatus("online-install"); err != nil {
				logger.Error(errors.Wrap(err, "failed to clear install task status"))
			}
			if err := setAppInstallState(pendingApp.ID, "installed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set app status to installed"))
			}
			if err := updatechecker.Configure(pendingApp.ID); err != nil {
				logger.Error(errors.Wrap(err, "failed to configure update checker"))
			}
		} else {
			if err := task.SetTaskStatus("online-install", finalError.Error(), "failed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set error on install task status"))
			}
			if err := setAppInstallState(pendingApp.ID, "install_error"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set app status to error"))
			}
		}
	}()

	db := persistence.MustGetPGSession()

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := task.SetTaskStatus("online-install", scanner.Text(), "running"); err != nil {
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
	query := `select id, title from cluster`
	rows, err := db.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query clusters")
	}
	defer rows.Close()

	clusterIDs := map[string]string{}
	for rows.Next() {
		clusterID := ""
		name := ""
		if err := rows.Scan(&clusterID, &name); err != nil {
			return nil, errors.Wrap(err, "failed to scan row")
		}

		clusterIDs[clusterID] = name
	}
	for clusterID, name := range clusterIDs {
		query = `insert into app_downstream (app_id, cluster_id, downstream_name) values ($1, $2, $3)`
		_, err = db.Exec(query, pendingApp.ID, clusterID, name)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create app downstream")
		}
	}

	query = `update app set is_airgap=false where id = $1`
	_, err = db.Exec(query, pendingApp.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update app to installed")
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

func setAppInstallState(appID string, status string) error {
	db := persistence.MustGetPGSession()

	query := `update app set install_state = $2 where id = $1`
	_, err := db.Exec(query, appID, status)
	if err != nil {
		return errors.Wrap(err, "failed to update app install state")
	}

	return nil
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
