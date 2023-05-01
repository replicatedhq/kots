package registry

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	containersmanifest "github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	containerstypes "github.com/containers/image/v5/types"
	"github.com/distribution/distribution/v3/reference"
	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

//go:embed assets/temp-registry-config.yml
var tempRegistryConfigYML string

type TempRegistry struct {
	process *os.Process
	port    string
}

// Start will spin up a docker registry service in the background on a random port.
// Will use the given "rootDir" as storage.
// Caller is responsible for stopping the registry.
func (r *TempRegistry) Start(rootDir string) (finalError error) {
	if r.port != "" {
		return errors.Errorf("registry is already running on port %s", r.port)
	}

	defer func() {
		if finalError != nil {
			r.Stop()
		}
	}()

	fp, err := freeport.GetFreePort()
	if err != nil {
		return errors.Wrap(err, "failed to get free port")
	}
	freePort := fmt.Sprintf("%d", fp)

	configYMLCopy := strings.Replace(tempRegistryConfigYML, "__ROOT_DIR__", rootDir, 1)
	configYMLCopy = strings.Replace(configYMLCopy, "__PORT__", freePort, 1)

	configFile, err := ioutil.TempFile("", "registryconfig")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file for config")
	}
	if err := ioutil.WriteFile(configFile.Name(), []byte(configYMLCopy), 0644); err != nil {
		return errors.Wrap(err, "failed to write config to temp file")
	}
	defer os.RemoveAll(configFile.Name())

	// We use the KOTS CLI as a wrapper to start the docker registry service because:
	// - We can't directly run the official docker registry binary because it doesn't necessarily exist when pushing images from the host.
	// - We need to be able to control stdout and stderr and stop the process later, but the registry go module doesn't give control over that.
	// - The KOTS CLI binary exists inside the kotsadm pod and/or will be used to push images from the host.
	cmd := exec.Command(kotsutil.GetKOTSBinPath(), "docker-registry", "serve", configFile.Name())
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start")
	}

	r.port = freePort
	r.process = cmd.Process

	if err := r.WaitForReady(time.Second * 30); err != nil {
		return errors.Wrap(err, "failed to wait for registry to become ready")
	}

	return nil
}

func (r *TempRegistry) Stop() {
	if r.process != nil {
		if err := r.process.Signal(os.Interrupt); err != nil {
			logger.Debugf("Failed to stop registry process on port %s", r.port)
		}
	}
	r.port = ""
	r.process = nil
}

func (r *TempRegistry) WaitForReady(timeout time.Duration) error {
	start := time.Now()

	for {
		url := fmt.Sprintf("http://localhost:%s", r.port)
		newRequest, err := http.NewRequest("GET", url, nil)
		if err == nil {
			resp, err := http.DefaultClient.Do(newRequest)
			if err == nil {
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}

		time.Sleep(time.Second)

		if time.Since(start) > timeout {
			return errors.Errorf("Timeout waiting for registry to become ready on port %s", r.port)
		}
	}
}

func (r *TempRegistry) GetImageLayers(image string) ([]types.Layer, error) {
	imageRef, err := reference.ParseDockerRef(image)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to normalize image %s", image)
	}

	tagOrDigest := ""
	if can, ok := imageRef.(reference.Canonical); ok {
		tagOrDigest = can.Digest().String()
	} else if tagged, ok := imageRef.(reference.Tagged); ok {
		tagOrDigest = tagged.Tag()
	} else {
		tagOrDigest = "latest"
	}

	imageParts := strings.Split(reference.TrimNamed(imageRef).Name(), "/") // strip tag and digest
	imageName := imageParts[len(imageParts)-1]                             // strip hostname and repo if any

	layers, err := r.getImageLayers(imageName, tagOrDigest)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get layers for image %s and tag/digest %s", imageName, tagOrDigest)
	}

	return layers, nil
}

func (r *TempRegistry) getImageLayers(imageName string, tagOrDigest string) ([]types.Layer, error) {
	url := fmt.Sprintf("http://localhost:%s/v2/%s/manifests/%s", r.port, imageName, tagOrDigest)
	newRequest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}

	for _, mediaType := range containersmanifest.DefaultRequestedManifestMIMETypes {
		newRequest.Header.Add("Accept", mediaType)
	}

	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute http request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code %d", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read server response")
	}

	layers := []types.Layer{}
	mimeType := containersmanifest.GuessMIMEType(b)

	if containersmanifest.MIMETypeIsMultiImage(mimeType) {
		// this is a multi-arch image, read layers for each architecture
		list, err := containersmanifest.ListFromBlob(b, mimeType)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list manifests from blob")
		}
		for _, digest := range list.Instances() {
			mLayers, err := r.getImageLayers(imageName, digest.String())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get layers for %s", digest.String())
			}
			layers = append(layers, mLayers...)
		}
	} else {
		manifest, err := containersmanifest.FromBlob(b, mimeType)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get manifest from blob")
		}
		for _, i := range manifest.LayerInfos() {
			if i.EmptyLayer {
				continue
			}
			layers = append(layers, types.Layer{
				Digest: i.Digest.String(),
				Size:   i.Size,
			})
		}
	}

	return layers, nil
}

func (r *TempRegistry) SrcRef(image string) (containerstypes.ImageReference, error) {
	parsed, err := reference.ParseDockerRef(image)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to normalize image %s", image)
	}
	normalizedImage := parsed.String()

	imageParts := strings.Split(normalizedImage, "/")
	lastPart := imageParts[len(imageParts)-1]

	srcRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://localhost:%s/%s", r.port, lastPart))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse source image name")
	}

	return srcRef, nil
}

// This is only used for integration tests
func (r *TempRegistry) OverridePort(port string) {
	r.port = port
}
