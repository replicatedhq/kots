package registry

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"path"
	"strings"
	"time"

	"github.com/containers/image/v5/docker"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	dockerregistry "github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/store"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
)

var deleteImagesTaskID = "delete-images"

func DeleteUnusedImages(ctx context.Context, registry types.RegistrySettings, usedImages []string) (finalError error) {
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
		return errors.Wrap(err, "failed to seacrh registry")
	}

	digestsInRegistry := map[string]string{}
	for _, r := range searchResult {
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
			taggedRef, err := docker.ParseReference(fmt.Sprintf("//%s", imageName))
			if err != nil {
				logger.Errorf("failed to parse tagged ref %q: %v", taggedName, err)
				continue
			}

			digest, err := docker.GetDigest(ctx, sysCtx, taggedRef)
			if err != nil {
				if !strings.Contains(err.Error(), "StatusCode: 404") {
					logger.Errorf("failed to get digest for %q: %v", taggedName, err)
				}
				continue
			}

			// Multiple image names can map to the same digest, but we only need to know one to delete the digest.
			digestsInRegistry[digest.String()] = taggedName
		}
	}

	for _, i := range usedImages {
		registryOptions := dockerregistry.RegistryOptions{
			Endpoint:  registry.Hostname,
			Namespace: registry.Namespace,
			Username:  registry.Username,
			Password:  registry.Password,
		}

		appImage := image.DestRef(registryOptions, i)
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
			return errors.Wrap(err, "failed to create remote executor")
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
			return errors.Wrap(err, "failed to stream command output")
		}

		return nil
	}

	return errors.New("no pods found to run garbage collect command")
}
