package registry

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/rewrite"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
)

// RewriteImages will use the app (a) and send the images to the registry specified. It will create patches for these
// and create a new version of the application
// the caller is responsible for deleting the appDir returned
func RewriteImages(appID string, sequence int64, hostname string, username string, password string, namespace string, isReadOnly bool) (appDir string, finalError error) {
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
	appDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}
	// appDir is returned

	err = store.GetStore().GetAppVersionArchive(appID, sequence, appDir)
	if err != nil {
		return "", errors.Wrap(err, "failed to get app version archive")
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

	installation, err := kotsutil.LoadInstallationFromPath(filepath.Join(appDir, "upstream", "userdata", "installation.yaml"))
	if err != nil {
		return "", errors.Wrap(err, "failed to load installation from path")
	}

	license, err := kotsutil.LoadLicenseFromPath(filepath.Join(appDir, "upstream", "userdata", "license.yaml"))
	if err != nil {
		return "", errors.Wrap(err, "failed to load license from path")
	}

	configValues, err := kotsutil.LoadConfigValuesFromFile(filepath.Join(appDir, "upstream", "userdata", "config.yaml"))
	if err != nil && !os.IsNotExist(errors.Cause(err)) {
		return "", errors.Wrap(err, "failed to load config values from path")
	}

	options := rewrite.RewriteOptions{
		RootDir:          appDir,
		UpstreamURI:      fmt.Sprintf("replicated://%s", license.Spec.AppSlug),
		UpstreamPath:     filepath.Join(appDir, "upstream"),
		Installation:     installation,
		Downstreams:      downstreamNames,
		CreateAppDir:     false,
		ExcludeKotsKinds: true,
		License:          license,
		ConfigValues:     configValues,
		K8sNamespace:     appNamespace,
		ReportWriter:     pipeWriter,
		IsAirgap:         a.IsAirgap,
		RegistrySettings: types.RegistrySettings{
			Hostname:   hostname,
			Namespace:  namespace,
			Username:   username,
			Password:   password,
			IsReadOnly: isReadOnly,
		},
		AppID:         a.ID,
		AppSlug:       a.Slug,
		IsGitOps:      a.IsGitOps,
		AppSequence:   nextAppSequence,
		ReportingInfo: reporting.GetReportingInfo(a.ID),

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
