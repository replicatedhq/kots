package image

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/containers/image/v5/copy"
	imagedocker "github.com/containers/image/v5/docker"
	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	containerstypes "github.com/containers/image/v5/types"
	"github.com/distribution/distribution/v3/reference"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/image/types"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/imageutil"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"golang.org/x/sync/errgroup"
)

var imagePolicy = []byte(`{
  "default": [{"type": "insecureAcceptAnything"}]
}`)

type UpdateInstallationImagesOptions struct {
	Images                 []string
	KotsKinds              *kotsutil.KotsKinds
	IsAirgap               bool
	UpstreamDir            string
	DockerHubRegistryCreds registry.Credentials
}

type UpdateInstallationAirgapArtifactsOptions struct {
	Artifacts   []string
	KotsKinds   *kotsutil.KotsKinds
	UpstreamDir string
}

func UpdateInstallationImages(opts UpdateInstallationImagesOptions) error {
	if opts.KotsKinds == nil {
		return nil
	}

	dockerHubRegistry := dockerregistrytypes.RegistryOptions{
		Username: opts.DockerHubRegistryCreds.Username,
		Password: opts.DockerHubRegistryCreds.Password,
	}

	installationImagesMap := make(map[string]imagetypes.InstallationImageInfo)
	for _, i := range opts.KotsKinds.Installation.Spec.KnownImages {
		installationImagesMap[i.Image] = imagetypes.InstallationImageInfo{
			IsPrivate: i.IsPrivate,
		}
	}

	allImagesPrivate := opts.IsAirgap
	if opts.KotsKinds.KotsApplication.Spec.ProxyPublicImages {
		allImagesPrivate = true
	}

	var mtx sync.Mutex
	const concurrencyLimit = 10
	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(concurrencyLimit)

	isPrivateImage := func(image string) (bool, error) {
		if allImagesPrivate {
			return true, nil
		}

		mtx.Lock()
		installationImage, ok := installationImagesMap[image]
		mtx.Unlock()
		if ok {
			return installationImage.IsPrivate, nil
		}

		p, err := IsPrivateImage(image, dockerHubRegistry)
		if err != nil {
			return false, err
		}
		return p, nil
	}

	for _, image := range opts.Images {
		func(image string) {
			g.Go(func() error {
				isPrivate, err := isPrivateImage(image)
				if err != nil {
					return errors.Wrapf(err, "failed to check if image %s is private", image)
				}
				mtx.Lock()
				installationImagesMap[image] = types.InstallationImageInfo{IsPrivate: isPrivate}
				mtx.Unlock()
				return nil
			})
		}(image)
	}

	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "failed to wait for image checks")
	}

	installationImages := []kotsv1beta1.InstallationImage{}
	for image, info := range installationImagesMap {
		installationImages = append(installationImages, kotsv1beta1.InstallationImage{
			Image:     image,
			IsPrivate: info.IsPrivate,
		})
	}

	// sort the images to get an ordered and reproducible output for easier testing
	sort.Slice(installationImages, func(i, j int) bool {
		return installationImages[i].Image < installationImages[j].Image
	})

	opts.KotsKinds.Installation.Spec.KnownImages = installationImages

	if err := kotsutil.SaveInstallation(&opts.KotsKinds.Installation, opts.UpstreamDir); err != nil {
		return errors.Wrap(err, "failed to save installation")
	}

	return nil
}

func UpdateInstallationAirgapArtifacts(opts UpdateInstallationAirgapArtifactsOptions) error {
	if opts.KotsKinds == nil {
		return nil
	}

	opts.KotsKinds.Installation.Spec.EmbeddedClusterArtifacts = opts.Artifacts

	if err := kotsutil.SaveInstallation(&opts.KotsKinds.Installation, opts.UpstreamDir); err != nil {
		return errors.Wrap(err, "failed to save installation")
	}

	return nil
}

func CopyOnlineImages(opts imagetypes.ProcessImageOptions, images []string, kotsKinds *kotsutil.KotsKinds, license *kotsv1beta1.License, dockerHubRegistryCreds registry.Credentials, log *logger.CLILogger) error {
	installationImages := make(map[string]imagetypes.InstallationImageInfo)
	for _, i := range kotsKinds.Installation.Spec.KnownImages {
		installationImages[i.Image] = imagetypes.InstallationImageInfo{
			IsPrivate: i.IsPrivate,
		}
	}

	replicatedRegistryInfo := registry.GetRegistryProxyInfo(license, &kotsKinds.Installation, &kotsKinds.KotsApplication)

	sourceRegistry := dockerregistrytypes.RegistryOptions{
		Endpoint:         replicatedRegistryInfo.Registry,
		ProxyEndpoint:    replicatedRegistryInfo.Proxy,
		UpstreamEndpoint: replicatedRegistryInfo.Upstream,
	}
	if license != nil {
		sourceRegistry.Username = license.Spec.LicenseID
		sourceRegistry.Password = license.Spec.LicenseID
	}

	dockerHubRegistry := dockerregistrytypes.RegistryOptions{
		Username: dockerHubRegistryCreds.Username,
		Password: dockerHubRegistryCreds.Password,
	}

	destRegistry := dockerregistrytypes.RegistryOptions{
		Endpoint:  opts.RegistrySettings.Hostname,
		Namespace: opts.RegistrySettings.Namespace,
		Username:  opts.RegistrySettings.Username,
		Password:  opts.RegistrySettings.Password,
	}

	copiedImages := map[string]bool{}
	for _, img := range images {
		if _, copied := copiedImages[img]; copied {
			continue
		}
		if err := copyOnlineImage(sourceRegistry, destRegistry, img, opts.AppSlug, opts.ReportWriter, log, installationImages, dockerHubRegistry); err != nil {
			return errors.Wrapf(err, "failed to copy online image %s", img)
		}
		copiedImages[img] = true
	}

	return nil
}

func copyOnlineImage(srcRegistry, destRegistry dockerregistrytypes.RegistryOptions, image string, appSlug string, reportWriter io.Writer, log *logger.CLILogger, installationImages map[string]types.InstallationImageInfo, dockerHubRegistry dockerregistrytypes.RegistryOptions) error {
	// TODO: This reaches out to internet in airgap installs. It shouldn't.
	sourceImage := image
	srcAuth := imagetypes.RegistryAuth{}
	if installationImages[image].IsPrivate {
		srcAuth = imagetypes.RegistryAuth{
			Username: srcRegistry.Username,
			Password: srcRegistry.Password,
		}
		rewritten, err := RewritePrivateImage(srcRegistry, image, appSlug)
		if err != nil {
			return errors.Wrap(err, "failed to rewrite private image")
		}
		sourceImage = rewritten
	}

	// normalize image to make sure only either a digest or a tag exist but not both
	parsedSrc, err := reference.ParseDockerRef(sourceImage)
	if err != nil {
		return errors.Wrapf(err, "failed to normalize source image %s", sourceImage)
	}
	sourceImage = parsedSrc.String()

	srcRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s", sourceImage))
	if err != nil {
		return errors.Wrapf(err, "failed to parse source image name %s", sourceImage)
	}

	destImage, err := imageutil.DestImage(destRegistry, image)
	if err != nil {
		return errors.Wrap(err, "failed to get destination image")
	}
	destStr := fmt.Sprintf("docker://%s", destImage)
	destRef, err := alltransports.ParseImageName(destStr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse dest image name %s", destStr)
	}

	copyAll := false
	if _, ok := parsedSrc.(reference.Canonical); ok {
		// we only support multi-arch images using digests
		copyAll = true
	}

	copyImageOpts := imagetypes.CopyImageOptions{
		SrcRef:  srcRef,
		DestRef: destRef,
		SrcAuth: srcAuth,
		DestAuth: imagetypes.RegistryAuth{
			Username: destRegistry.Username,
			Password: destRegistry.Password,
		},
		CopyAll:           copyAll,
		SrcDisableV1Ping:  true,
		SrcSkipTLSVerify:  os.Getenv("KOTSADM_INSECURE_SRCREGISTRY") == "true",
		DestDisableV1Ping: true,
		DestSkipTLSVerify: true,
		ReportWriter:      reportWriter,
	}
	if err := CopyImage(copyImageOpts); err != nil {
		return errors.Wrapf(err, "failed to copy %s to %s", sourceImage, destImage)
	}

	return nil
}

func CopyImage(opts types.CopyImageOptions) error {
	srcCtx := &containerstypes.SystemContext{}
	destCtx := &containerstypes.SystemContext{}

	if opts.SrcDisableV1Ping {
		srcCtx.DockerDisableV1Ping = true
	}
	if opts.SrcSkipTLSVerify {
		srcCtx.DockerInsecureSkipTLSVerify = containerstypes.OptionalBoolTrue
	}
	if opts.DestDisableV1Ping {
		destCtx.DockerDisableV1Ping = true
	}
	if opts.DestSkipTLSVerify {
		destCtx.DockerInsecureSkipTLSVerify = containerstypes.OptionalBoolTrue
	}

	if opts.SrcAuth.Username != "" && opts.SrcAuth.Password != "" {
		srcCtx.DockerAuthConfig = &containerstypes.DockerAuthConfig{
			Username: opts.SrcAuth.Username,
			Password: opts.SrcAuth.Password,
		}
	}

	username, password := opts.DestAuth.Username, opts.DestAuth.Password
	registryHost := reference.Domain(opts.DestRef.DockerReference())

	if registry.IsECREndpoint(registryHost) && username != "AWS" {
		login, err := registry.GetECRLogin(registryHost, username, password)
		if err != nil {
			return errors.Wrap(err, "failed to get ECR login")
		}
		username = login.Username
		password = login.Password
	}

	if username != "" && password != "" {
		destCtx.DockerAuthConfig = &containerstypes.DockerAuthConfig{
			Username: username,
			Password: password,
		}
	}

	imageListSelection := copy.CopySystemImage
	if opts.CopyAll {
		imageListSelection = copy.CopyAllImages
	}

	_, err := CopyImageWithGC(context.Background(), opts.DestRef, opts.SrcRef, &copy.Options{
		RemoveSignatures:      true,
		SignBy:                "",
		ReportWriter:          opts.ReportWriter,
		SourceCtx:             srcCtx,
		DestinationCtx:        destCtx,
		ForceManifestMIMEType: "",
		ImageListSelection:    imageListSelection,
	})
	if err != nil {
		return errors.Wrap(err, "failed to copy image")
	}

	return nil
}

// if dockerHubRegistry is provided, its credentials will be used for DockerHub images to increase the rate limit.
func IsPrivateImage(image string, dockerHubRegistry dockerregistrytypes.RegistryOptions) (bool, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		dockerRef, err := dockerref.ParseDockerRef(image)
		if err != nil {
			return false, errors.Wrapf(err, "failed to parse docker ref %q", image)
		}

		sysCtx := containerstypes.SystemContext{DockerDisableV1Ping: true}

		registryHost := reference.Domain(dockerRef)
		isDockerIO := registryHost == "docker.io" || strings.HasSuffix(registryHost, ".docker.io")
		if isDockerIO && dockerHubRegistry.Username != "" && dockerHubRegistry.Password != "" {
			sysCtx.DockerAuthConfig = &containerstypes.DockerAuthConfig{
				Username: dockerHubRegistry.Username,
				Password: dockerHubRegistry.Password,
			}
		}

		// allow pulling images from http/invalid https docker repos
		// intended for development only, _THIS MAKES THINGS INSECURE_
		if os.Getenv("KOTSADM_INSECURE_SRCREGISTRY") == "true" {
			sysCtx.DockerInsecureSkipTLSVerify = containerstypes.OptionalBoolTrue
		}

		// ParseReference requires the // prefix
		imageRef, err := imagedocker.ParseReference(fmt.Sprintf("//%s", dockerRef.String()))
		if err != nil {
			return false, errors.Wrapf(err, "failed to parse image ref %s", dockerRef.String())
		}

		remoteImage, err := imageRef.NewImageSource(context.Background(), &sysCtx)
		if err == nil {
			remoteImage.Close()
			return false, nil
		}

		// manifest was downloaded, but no matching architecture found in manifest.
		// still, not a private image
		if strings.Contains(err.Error(), "no image found in manifest list for architecture") {
			return false, nil
		}

		if strings.Contains(err.Error(), "EOF") {
			lastErr = err
			time.Sleep(1 * time.Second)
			continue
		}

		logger.Infof("Marking image '%s' as private because: %v", image, err.Error())

		// if the registry is unreachable (which might be due to a firewall, proxy, etc..),
		// we won't be able to determine if the error is due to a missing auth or not.
		// so we consider the image private. a use-case for this is when the images are supposed to be
		// proxied through proxy.replicated.com and the other domains are blocked by the firewall.

		return true, nil
	}

	return false, errors.Wrap(lastErr, "failed to retry")
}

func RewritePrivateImage(srcRegistry dockerregistrytypes.RegistryOptions, image string, appSlug string) (string, error) {
	dockerRef, err := dockerref.ParseDockerRef(image)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse docker ref %q", image)
	}

	registryHost := dockerref.Domain(dockerRef)
	if registryHost == srcRegistry.Endpoint {
		// replicated images are also private, but we don't rewrite those
		return image, nil
	}

	if registryHost == srcRegistry.UpstreamEndpoint {
		// image is using the upstream replicated registry, but a custom registry domain is configured, so rewrite to use the custom domain
		return strings.Replace(image, registryHost, srcRegistry.Endpoint, 1), nil
	}

	newImage := registry.MakeProxiedImageURL(srcRegistry.ProxyEndpoint, appSlug, image)
	if can, ok := dockerRef.(reference.Canonical); ok {
		return newImage + "@" + can.Digest().String(), nil
	} else if tagged, ok := dockerRef.(dockerref.Tagged); ok {
		return newImage + ":" + tagged.Tag(), nil
	}

	// no tag, so it will be "latest"
	return newImage, nil
}

func getPolicyContext() (*signature.PolicyContext, error) {
	policy, err := signature.NewPolicyFromBytes(imagePolicy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read default policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create policy")
	}
	return policyContext, nil
}

func CopyImageWithGC(ctx context.Context, destRef, srcRef containerstypes.ImageReference, options *copy.Options) ([]byte, error) {
	policyContext, err := getPolicyContext()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get policy")
	}

	manifest, err := copy.Image(ctx, policyContext, destRef, srcRef, options)

	// copying an image increases allocated memory, which can push the pod to cross the memory limit when copying multiple images in a row.
	runtime.GC()

	return manifest, err
}

func FindImagesInDir(dir string) ([]string, error) {
	uniqueImages := make(map[string]bool)

	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				return errors.Wrapf(err, "failed to read file %s", path)
			}

			return listImagesInFile(contents, func(images []string, doc k8sdoc.K8sDoc) error {
				for _, image := range images {
					uniqueImages[image] = true
				}
				return nil
			})
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk dir")
	}

	result := make([]string, 0, len(uniqueImages))
	for i := range uniqueImages {
		result = append(result, i)
	}
	sort.Strings(result) // sort the images to get an ordered and reproducible output for easier testing

	return result, nil
}

type processImagesFunc func([]string, k8sdoc.K8sDoc) error

func listImagesInFile(contents []byte, handler processImagesFunc) error {
	yamlDocs := util.ConvertToSingleDocs(contents)
	for _, yamlDoc := range yamlDocs {
		parsed, err := k8sdoc.ParseYAML(yamlDoc)
		if err != nil {
			continue
		}

		images := parsed.ListImages()

		if err := handler(images, parsed); err != nil {
			return err
		}
	}

	return nil
}
