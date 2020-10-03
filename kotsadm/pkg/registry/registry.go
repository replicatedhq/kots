package registry

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/reporting"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/rewrite"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// RewriteImages will use the app (a) and send the images to the registry specified. It will create patches for these
// and create a new version of the application
// the caller is responsible for deleting the appDir returned
func RewriteImages(appID string, sequence int64, hostname string, username string, password string, namespace string, configValues *kotsv1beta1.ConfigValues) (appDir string, finalError error) {
	if err := store.GetStore().SetTaskStatus("image-rewrite", "Updating registry settings", "running"); err != nil {
		return "", errors.Wrap(err, "failed to set task status")
	}

	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := store.GetStore().UpdateTaskStatusTimestamp("image-rewrite"); err != nil {
					logger.Error(err)
				}
			case <-finishedCh:
				return
			}
		}
	}()

	defer func() {
		if finalError == nil {
			if err := store.GetStore().ClearTaskStatus("image-rewrite"); err != nil {
				logger.Error(err)
			}
		} else {
			// do not show the stack trace to the user
			causeErr := errors.Cause(finalError)
			if err := store.GetStore().SetTaskStatus("image-rewrite", causeErr.Error(), "failed"); err != nil {
				logger.Error(err)
			}
		}
	}()

	// get the archive and store it in a temporary location
	appDir, err := store.GetStore().GetAppVersionArchive(appID, sequence)
	if err != nil {
		return "", errors.Wrap(err, "failed to get app version archive")
	}
	defer os.RemoveAll(appDir)

	installation, err := kotsutil.LoadInstallationFromPath(filepath.Join(appDir, "upstream", "userdata", "installation.yaml"))
	if err != nil {
		return "", errors.Wrap(err, "failed to load installation from path")
	}

	license, err := kotsutil.LoadLicenseFromPath(filepath.Join(appDir, "upstream", "userdata", "license.yaml"))
	if err != nil {
		return "", errors.Wrap(err, "failed to load license from path")
	}

	if configValues == nil {
		previousConfigValues, err := kotsutil.LoadConfigValuesFromFile(filepath.Join(appDir, "upstream", "userdata", "config.yaml"))
		if err != nil && !os.IsNotExist(errors.Cause(err)) {
			return "", errors.Wrap(err, "failed to load config values from path")
		}

		configValues = previousConfigValues
	}

	// get the downstream names only
	downstreams, err := store.GetStore().ListDownstreamsForApp(appID)
	if err != nil {
		return "", errors.Wrap(err, "failed to list downstreams")
	}

	downstreamNames := []string{}
	for _, d := range downstreams {
		downstreamNames = append(downstreamNames, d.Name)
	}

	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get app")
	}

	// dev_namespace makes the dev env work
	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := store.GetStore().SetTaskStatus("image-rewrite", scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	options := rewrite.RewriteOptions{
		RootDir:           appDir,
		UpstreamURI:       fmt.Sprintf("replicated://%s", license.Spec.AppSlug),
		UpstreamPath:      filepath.Join(appDir, "upstream"),
		Installation:      installation,
		Downstreams:       downstreamNames,
		CreateAppDir:      false,
		ExcludeKotsKinds:  true,
		License:           license,
		ConfigValues:      configValues,
		K8sNamespace:      appNamespace,
		ReportWriter:      pipeWriter,
		CopyImages:        true,
		IsAirgap:          a.IsAirgap,
		RegistryEndpoint:  hostname,
		RegistryUsername:  username,
		RegistryPassword:  password,
		RegistryNamespace: namespace,
		AppSlug:           a.Slug,
		IsGitOps:          a.IsGitOps,
		AppSequence:       a.CurrentSequence + 1, // sequence +1 because this is the current latest sequence, not the sequence that the rendered version will be saved as
		ReportingInfo:     reporting.GetReportingInfo(a.ID),
	}

	if err := rewrite.Rewrite(options); err != nil {
		return "", errors.Wrap(err, "failed to rewrite images")
	}

	return appDir, nil
}

func HasKurlRegistry() (bool, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return false, errors.Wrap(err, "failed to get config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return false, errors.Wrap(err, "failed to create clientset")
	}

	registryCredsSecret, err := clientset.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.TODO(), "registry-creds", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		// this is not an error, it could be rbac
		// don't even log it, normal operations
		return false, nil
	}

	if registryCredsSecret != nil {
		if registryCredsSecret.Type == corev1.SecretTypeDockerConfigJson {
			return true, nil
		}
	}

	return false, nil
}

func GetKotsadmRegistry() (*types.RegistrySettings, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client set")
	}

	namespace := os.Getenv("POD_NAMESPACE")

	kotsadmOptions, err := kotsadm.GetKotsadmOptionsFromCluster(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm options from cluster")
	}

	registry := kotsadmOptions.OverrideRegistry
	registryNamespace := kotsadmOptions.OverrideNamespace
	hostParts := strings.Split(kotsadmOptions.OverrideRegistry, "/")
	if len(hostParts) == 2 {
		registry = hostParts[0]
		registryNamespace = hostParts[1]
	}

	registrySettings := types.RegistrySettings{
		Hostname:  registry,
		Namespace: registryNamespace,
		Username:  kotsadmOptions.Username,
		Password:  kotsadmOptions.Password,
	}

	return &registrySettings, nil
}
