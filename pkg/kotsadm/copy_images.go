package kotsadm

import (
	"context"
	"fmt"
	"os"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/transports/alltransports"
	imagev5types "github.com/containers/image/v5/types"
	"github.com/distribution/distribution/v3/reference"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"k8s.io/client-go/kubernetes"
)

// Copies Admin Console images from public registry to private registry
func CopyImages(options imagetypes.PushImagesOptions, kotsNamespace string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	// Minimal info needed to get the right image names
	deployOptions := kotsadmtypes.DeployOptions{
		IsOpenShift: k8sutil.IsOpenShift(clientset),
		RegistryConfig: kotsadmtypes.RegistryConfig{
			OverrideRegistry:  options.Registry.Endpoint,
			OverrideNamespace: options.Registry.Namespace,
			Username:          options.Registry.Username,
			Password:          options.Registry.Password,
		},
	}

	sourceImages := kotsadmobjects.GetOriginalAdminConsoleImages(deployOptions)
	destImages := kotsadmobjects.GetAdminConsoleImages(deployOptions)
	for imageName, sourceImage := range sourceImages {
		destImage := destImages[imageName]
		if destImage == "" {
			return errors.Errorf("failed to find image %s in destination list", imageName)
		}

		image.WriteProgressLine(options.ProgressWriter, fmt.Sprintf("Copying %s to %s", sourceImage, destImage))

		sourceCtx, err := getCopyImagesSourceContext(clientset, kotsNamespace)
		if err != nil {
			return errors.Wrap(err, "failed to get source context")
		}

		srcRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s", sourceImage))
		if err != nil {
			return errors.Wrapf(err, "failed to parse source image name %s", sourceImage)
		}

		destStr := fmt.Sprintf("docker://%s", destImage)
		destRef, err := alltransports.ParseImageName(destStr)
		if err != nil {
			return errors.Wrapf(err, "failed to parse dest image name %s", destStr)
		}

		destCtx := &imagev5types.SystemContext{
			DockerInsecureSkipTLSVerify: imagev5types.OptionalBoolTrue,
			DockerDisableV1Ping:         true,
		}

		username, password := options.Registry.Username, options.Registry.Password
		registryHost := reference.Domain(destRef.DockerReference())

		if registry.IsECREndpoint(registryHost) && username != "AWS" {
			login, err := registry.GetECRLogin(registryHost, username, password)
			if err != nil {
				return errors.Wrap(err, "failed to get ECR login")
			}
			username = login.Username
			password = login.Password
		}

		if username != "" && password != "" {
			destCtx.DockerAuthConfig = &imagev5types.DockerAuthConfig{
				Username: username,
				Password: password,
			}
		}

		_, err = image.CopyImageWithGC(context.Background(), destRef, srcRef, &copy.Options{
			RemoveSignatures:      true,
			SignBy:                "",
			ReportWriter:          options.ProgressWriter,
			SourceCtx:             sourceCtx,
			DestinationCtx:        destCtx,
			ForceManifestMIMEType: "",
		})
		if err != nil {
			return errors.Wrapf(err, "failed to copy %s to %s: %v", sourceImage, destImage, err)
		}
	}

	return nil
}

func getCopyImagesSourceContext(clientset kubernetes.Interface, kotsNamespace string) (*imagev5types.SystemContext, error) {
	sourceCtx := &imagev5types.SystemContext{DockerDisableV1Ping: true}

	// allow pulling images from http/invalid https docker repos
	// intended for development only, _THIS MAKES THINGS INSECURE_
	if os.Getenv("KOTSADM_INSECURE_SRCREGISTRY") == "true" {
		sourceCtx.DockerInsecureSkipTLSVerify = imagev5types.OptionalBoolTrue
	}

	credentials, err := registry.GetDockerHubCredentials(clientset, kotsNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get docker hub credentials")
	}

	if credentials.Username != "" && credentials.Password != "" {
		sourceCtx.DockerAuthConfig = &imagev5types.DockerAuthConfig{
			Username: credentials.Username,
			Password: credentials.Password,
		}
	}

	return sourceCtx, nil
}
