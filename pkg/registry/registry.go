package registry

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/rewrite"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RewriteImages will use the app (a) and send the images to the registry specified. It will create patches for these
// and create a new version of the application
// the caller is responsible for deleting the appDir returned
func RewriteImages(appID string, sequence int64, hostname string, username string, password string, namespace string, isReadOnly bool, configValues *kotsv1beta1.ConfigValues) (appDir string, finalError error) {
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
				logger.Error(errors.Wrap(err, "failed to clear image rewrite task status"))
			}
		} else {
			// do not show the stack trace to the user
			causeErr := errors.Cause(finalError)
			if err := store.GetStore().SetTaskStatus("image-rewrite", causeErr.Error(), "failed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set image rewrite task status as failed"))
			}
		}
	}()

	// get the archive and store it in a temporary location
	appDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}
	// appDir is returned

	err = store.GetStore().GetAppVersionArchive(appID, sequence, appDir)
	if err != nil {
		return "", errors.Wrap(err, "failed to get app version archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(appDir)
	if err != nil {
		return "", errors.Wrap(err, "failed to load kotskinds from path")
	}

	if configValues == nil {
		configValues = kotsKinds.ConfigValues
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
	appNamespace := util.PodNamespace
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

	nextAppSequence, err := store.GetStore().GetNextAppSequence(a.ID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get next app sequence")
	}

	options := rewrite.RewriteOptions{
		RootDir:            appDir,
		UpstreamURI:        fmt.Sprintf("replicated://%s", kotsKinds.License.Spec.AppSlug),
		UpstreamPath:       filepath.Join(appDir, "upstream"),
		Installation:       &kotsKinds.Installation,
		Downstreams:        downstreamNames,
		CreateAppDir:       false,
		ExcludeKotsKinds:   true,
		License:            kotsKinds.License,
		ConfigValues:       configValues,
		KotsApplication:    &kotsKinds.KotsApplication,
		K8sNamespace:       appNamespace,
		ReportWriter:       pipeWriter,
		IsAirgap:           a.IsAirgap,
		RegistryEndpoint:   hostname,
		RegistryUsername:   username,
		RegistryPassword:   password,
		RegistryNamespace:  namespace,
		RegistryIsReadOnly: isReadOnly,
		AppID:              a.ID,
		AppSlug:            a.Slug,
		IsGitOps:           a.IsGitOps,
		AppSequence:        nextAppSequence,
		ReportingInfo:      reporting.GetReportingInfo(a.ID),

		// TODO: pass in as arguments if this is ever called from CLI
		HTTPProxyEnvValue:  os.Getenv("HTTP_PROXY"),
		HTTPSProxyEnvValue: os.Getenv("HTTPS_PROXY"),
		NoProxyEnvValue:    os.Getenv("NO_PROXY"),
	}

	options.CopyImages = true
	if isReadOnly {
		options.CopyImages = false
	}

	if err := rewrite.Rewrite(options); err != nil {
		return "", errors.Wrap(err, "failed to rewrite images")
	}

	return appDir, nil
}

func HasKurlRegistry() (bool, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return false, errors.Wrap(err, "failed to get k8s clientset")
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
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	namespace := util.PodNamespace

	registryConfig, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm options from cluster")
	}

	registry := registryConfig.OverrideRegistry
	registryNamespace := registryConfig.OverrideNamespace
	hostParts := strings.Split(registryConfig.OverrideRegistry, "/")
	if len(hostParts) == 2 {
		registry = hostParts[0]
		registryNamespace = hostParts[1]
	}

	registrySettings := types.RegistrySettings{
		Hostname:   registry,
		Namespace:  registryNamespace,
		Username:   registryConfig.Username,
		Password:   registryConfig.Password,
		IsReadOnly: registryConfig.IsReadOnly,
	}

	return &registrySettings, nil
}
