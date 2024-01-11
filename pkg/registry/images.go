package registry

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/containers/image/v5/docker"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/imageutil"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/registry/types"
	kotss3 "github.com/replicatedhq/kots/pkg/s3"
	"github.com/replicatedhq/kots/pkg/store"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
)

var deleteImagesTaskID = "delete-images"

type AppRollbackError struct {
	AppID    string
	Sequence int64
}

func (e AppRollbackError) Error() string {
	return fmt.Sprintf("app:%s, version:%d", e.AppID, e.Sequence)
}

func shouldGarbageCollectImages(isKurl bool, kurlRegistryHost string, installParams kotsutil.InstallationParams, registrySettings types.RegistrySettings) bool {
	if !installParams.EnableImageDeletion {
		logger.Info("ignoring image garbage collection because image deletion is disabled")
		return false
	}

	if registrySettings.IsReadOnly {
		logger.Info("ignoring image garbage collection because registry is read only")
		return false
	}

	if !isKurl {
		logger.Info("ignoring image garbage collection because cluster is not kurl")
		return false
	}

	if kurlRegistryHost != registrySettings.Hostname {
		logger.Info("ignoring image garbage collection because registry is not kurl registry")
		return false
	}

	return true
}

func DeleteUnusedImages(appID string, ignoreRollback bool) error {
	installParams, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		return errors.Wrap(err, "failed to get app registry info")
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app registry info")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	isKurl, err := kurl.IsKurl(clientset)
	if err != nil {
		return errors.Wrap(err, "failed to check if cluster is kurl")
	}

	kurlRegistryHost, _, _, err := kotsutil.GetKurlRegistryCreds()
	if err != nil {
		return errors.Wrap(err, "failed to get kurl registry creds")
	}

	if !shouldGarbageCollectImages(isKurl, kurlRegistryHost, installParams, registrySettings) {
		return nil
	}

	// we check all apps here because different apps could share the same images,
	// and the images could be active in one but not the other.
	// so, we also do not delete the images if rollback is enabled for any app.
	appIDs, err := store.GetStore().GetAppIDsFromRegistry(registrySettings.Hostname)
	if err != nil {
		return errors.Wrap(err, "failed to get apps with registry")
	}

	activeVersions := []*downstreamtypes.DownstreamVersion{}
	for _, appID := range appIDs {
		a, err := store.GetStore().GetApp(appID)
		if err != nil {
			errors.Wrap(err, "failed to get app")
		}

		if !ignoreRollback {
			// rollback support is detected from the latest available version, not the currently deployed one
			latestSequence, err := store.GetStore().GetLatestAppSequence(a.ID, true)
			if err != nil {
				return errors.Wrap(err, "failed to get latest app sequence")
			}
			allowRollback, err := store.GetStore().IsRollbackSupportedForVersion(a.ID, latestSequence)
			if err != nil {
				return errors.Wrap(err, "failed to check if rollback is supported")
			}
			if allowRollback {
				return AppRollbackError{AppID: a.ID, Sequence: latestSequence}
			}
		} else {
			logger.Info("ignoring the fact that rollback is enabled and will continue with the images removal process")
		}

		downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
		if err != nil {
			return errors.Wrap(err, "failed to list downstreams for app")
		}

		for _, d := range downstreams {
			downstreamVersions, err := store.GetStore().GetDownstreamVersions(a.ID, d.ClusterID, false)
			if err != nil {
				return errors.Wrapf(err, "failed to get app versions for downstream %s", d.ClusterID)
			}

			// current version already has additional details, get details for pending versions
			if err := store.GetStore().AddDownstreamVersionsDetails(a.ID, d.ClusterID, downstreamVersions.PendingVersions, false); err != nil {
				return errors.Wrapf(err, "failed to add details for pending versions for downstream %s", d.ClusterID)
			}

			activeVersions = append(activeVersions, downstreamVersions.CurrentVersion)
			activeVersions = append(activeVersions, downstreamVersions.PendingVersions...)
		}
	}

	imagesDedup := map[string]struct{}{}
	for _, version := range activeVersions {
		if version == nil {
			continue
		}
		if version.KOTSKinds == nil {
			continue
		}
		for _, i := range version.KOTSKinds.Installation.Spec.KnownImages {
			imagesDedup[i.Image] = struct{}{}
		}
	}

	usedImages := []string{}
	for i, _ := range imagesDedup {
		usedImages = append(usedImages, i)
	}

	if installParams.KotsadmRegistry != "" {
		deployOptions := kotsadmtypes.DeployOptions{
			// Minimal info needed to get the right image names
			RegistryConfig: kotsadmtypes.RegistryConfig{
				// TODO: OverrideVersion
				OverrideRegistry:  registrySettings.Hostname,
				OverrideNamespace: registrySettings.Namespace,
				Username:          registrySettings.Username,
				Password:          registrySettings.Password,
			},
		}
		kotsadmImages := kotsadmobjects.GetAdminConsoleImages(deployOptions)
		for _, i := range kotsadmImages {
			usedImages = append(usedImages, i)
		}
	}

	err = deleteUnusedImages(context.Background(), registrySettings, usedImages)
	if err != nil {
		return errors.Wrap(err, "failed to delete unused images")
	}

	return nil
}

func deleteUnusedImages(ctx context.Context, registry types.RegistrySettings, usedImages []string) (finalError error) {
	if registry.Hostname == "" {
		return nil
	}

	currentStatus, _, err := store.GetStore().GetTaskStatus(deleteImagesTaskID)
	if err != nil {
		return errors.Wrap(err, "failed to get task status")
	}

	if currentStatus == "running" {
		logger.Debugf("%s is already running, not starting a new one", deleteImagesTaskID)
		return nil
	}

	if err := store.GetStore().SetTaskStatus(deleteImagesTaskID, "Searching registry...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	finishedChan := make(chan error)
	defer close(finishedChan)

	startDeleteImagesTaskMonitor(finishedChan)
	defer func() {
		finishedChan <- finalError
	}()

	sysCtx := &imagetypes.SystemContext{
		DockerInsecureSkipTLSVerify: imagetypes.OptionalBoolTrue,
		DockerDisableV1Ping:         true,
	}
	if registry.Username != "" && registry.Password != "" {
		sysCtx.DockerAuthConfig = &imagetypes.DockerAuthConfig{
			Username: registry.Username,
			Password: registry.Password,
		}
	}

	searchResult, err := docker.SearchRegistry(ctx, sysCtx, registry.Hostname, "", math.MaxInt32)
	if err != nil {
		return errors.Wrap(err, "failed to search registry")
	}

	digestsInRegistry := map[string]string{}
	for _, r := range searchResult {
		// the registry can be shared with other internal or external applications, specially if an external registry is configured.
		// ONLY delete images from the configured application's registry namespace to avoid deleting non-related user data.
		parts := strings.Split(r.Name, "/")
		registryNamespace := ""
		if len(parts) > 1 {
			// e.g.: my/namespace/imagename => my/namespace
			registryNamespace = path.Join(parts[:len(parts)-1]...)
		}
		if registryNamespace != registry.Namespace {
			continue
		}

		imageName := path.Join(registry.Hostname, r.Name)

		ref, err := docker.ParseReference(fmt.Sprintf("//%s", imageName))
		if err != nil {
			logger.Errorf("failed to parse registry image ref %q: %v", imageName, err)
			continue
		}

		tags, err := docker.GetRepositoryTags(ctx, sysCtx, ref)
		if err != nil {
			logger.Errorf("failed to get repo tags for %q: %v", imageName, err)
			continue
		}

		for _, tag := range tags {
			taggedName := fmt.Sprintf("%s:%s", imageName, tag)
			taggedRef, err := docker.ParseReference(fmt.Sprintf("//%s", taggedName))
			if err != nil {
				logger.Errorf("failed to parse tagged ref %q: %v", taggedName, err)
				continue
			}

			digest, err := docker.GetDigest(ctx, sysCtx, taggedRef)
			if err != nil {
				if !strings.Contains(err.Error(), "StatusCode: 404") {
					logger.Errorf("failed to get digest for %q: %v", taggedName, err)
				} else {
					logger.Infof("will not delete %q it's not found in registry", taggedName)
				}
				continue
			}

			// Multiple image names can map to the same digest, but we only need to know one to delete the digest.
			digestsInRegistry[digest.String()] = taggedName
		}
	}

	for _, usedImage := range usedImages {
		registryOptions := registrytypes.RegistryOptions{
			Endpoint:  registry.Hostname,
			Namespace: registry.Namespace,
			Username:  registry.Username,
			Password:  registry.Password,
		}

		appImage, err := imageutil.DestImage(registryOptions, usedImage)
		if err != nil {
			return errors.Wrapf(err, "failed to get destination image for %s", appImage)
		}

		appImageRef, err := docker.ParseReference(fmt.Sprintf("//%s", appImage))
		if err != nil {
			return errors.Wrapf(err, "failed to parse %s", appImage)
		}

		digest, err := docker.GetDigest(ctx, sysCtx, appImageRef)
		if err != nil {
			if !strings.Contains(err.Error(), "StatusCode: 404") {
				return errors.Wrapf(err, "failed to get digest for %s", appImage)
			}
			logger.Infof("digest not found for image %q", appImage)
			continue
		}

		delete(digestsInRegistry, digest.String())
	}

	for digest, imageName := range digestsInRegistry {
		logger.Infof("Deleting digest %s for image %s", digest, imageName)
		ref, err := docker.ParseReference(fmt.Sprintf("//%s", imageName))
		if err != nil {
			logger.Infof("failed to parse image ref %q: %v", imageName, err)
			continue
		}

		err = ref.DeleteImage(ctx, sysCtx)
		if err != nil {
			logger.Infof("failed to delete image %q from registry: %v\n", imageName, err)
			continue
		}
	}

	if err := runGCCommand(ctx); err != nil {
		return errors.Wrap(err, "failed to run garbage collect command")
	}

	return nil
}

func startDeleteImagesTaskMonitor(finishedChan <-chan error) {
	go func() {
		var finalError error
		defer func() {
			if finalError == nil {
				if err := store.GetStore().ClearTaskStatus(deleteImagesTaskID); err != nil {
					logger.Error(errors.Wrapf(err, "failed to clear %q task status", deleteImagesTaskID))
				}
			} else {
				if err := store.GetStore().SetTaskStatus(deleteImagesTaskID, finalError.Error(), "failed"); err != nil {
					logger.Error(errors.Wrapf(err, "failed to set error on %q task status", deleteImagesTaskID))
				}
			}
		}()

		for {
			select {
			case <-time.After(time.Second):
				if err := store.GetStore().UpdateTaskStatusTimestamp(deleteImagesTaskID); err != nil {
					logger.Error(err)
				}
			case err := <-finishedChan:
				finalError = err
				return
			}
		}
	}()
}

func runGCCommand(ctx context.Context) error {
	clusterConfig, err := k8sutil.GetClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return errors.Wrap(err, "failed runtime to scheme")
	}

	registryPods, err := clientset.CoreV1().Pods("kurl").List(ctx, metav1.ListOptions{LabelSelector: "app=registry"})
	if err != nil {
		return errors.Wrap(err, "failed to list registry pods")
	}

	// let's create an empty file named "empty" in a well-known location to work around a bug in how ceph and the registry
	// operate together: https://github.com/goharbor/harbor/issues/11929#issuecomment-828892005
	// we don't care if this file exists, so just ignore errors for now
	_ = uploadEmptyFileToRegistry(ctx)

	errs := make([]error, 0)
	for _, pod := range registryPods.Items {
		req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(pod.Name).Namespace(pod.Namespace).SubResource("exec")
		parameterCodec := runtime.NewParameterCodec(scheme)
		req.VersionedParams(&corev1.PodExecOptions{
			Command:   []string{"/bin/registry", "garbage-collect", "/etc/docker/registry/config.yml"},
			Container: "registry",
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, parameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(clusterConfig, "POST", req.URL())
		if err != nil {
			errs = append(errs, errors.Wrap(err, "failed to create remote executor"))
			continue
		}

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)

		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  nil,
			Stdout: stdout,
			Stderr: stderr,
			Tty:    false,
		})

		logger.Infof("garbage collect command stdout: %s", stdout.Bytes())
		logger.Infof("garbage collect command stderr: %s", stderr.Bytes())

		if err != nil {
			errs = append(errs, errors.Wrap(err, "failed to stream command output"))
			continue
		}

		// terminate after the first successful loop iteration
		return nil
	}

	logger.Errorf("errors while running garbage collect command: %v", errs)
	return errors.New("no pods found to run garbage collect command")
}

func uploadEmptyFileToRegistry(ctx context.Context) error {
	bucketName := "docker-registry"
	contents := []byte("")
	path := filepath.Join("docker", "registry", "v2", "repositories", "empty")

	newSession := awssession.New(kotss3.GetConfig())
	s3Client := s3.New(newSession)

	_, err := s3Client.PutObject(&s3.PutObjectInput{
		Body:   bytes.NewReader(contents),
		Bucket: aws.String(bucketName),
		Key:    aws.String(path),
	})

	return err
}
