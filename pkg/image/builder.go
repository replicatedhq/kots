package image

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
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
	dockertypes "github.com/replicatedhq/kots/pkg/docker/types"
	"github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	regsitrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"golang.org/x/sync/errgroup"
	kustomizeimage "sigs.k8s.io/kustomize/api/types"
)

var imagePolicy = []byte(`{
  "default": [{"type": "insecureAcceptAnything"}]
}`)

func RewriteImages(srcRegistry, destRegistry dockerregistrytypes.RegistryOptions, appSlug string, log *logger.CLILogger, reportWriter io.Writer, upstreamDir string, additionalImages []string, copyImages, allImagesPrivate bool, checkedImages map[string]types.ImageInfo, dockerHubRegistry dockerregistrytypes.RegistryOptions) ([]kustomizeimage.Image, error) {
	newImages := []kustomizeimage.Image{}
	savedImages := map[string]bool{}

	err := filepath.Walk(upstreamDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			contents, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			newImagesSubset, err := rewriteImagesInFileBetweenRegistries(srcRegistry, destRegistry, appSlug, log, reportWriter, contents, copyImages, allImagesPrivate, checkedImages, savedImages, dockerHubRegistry)
			if err != nil {
				return errors.Wrapf(err, "failed to copy images mentioned in %s", path)
			}

			newImages = append(newImages, newImagesSubset...)
			return nil
		})

	if err != nil {
		return nil, errors.Wrap(err, "failed to walk upstream dir")
	}

	for _, additionalImage := range additionalImages {
		newImage, err := rewriteOneImage(srcRegistry, destRegistry, additionalImage, appSlug, reportWriter, log, copyImages, allImagesPrivate, checkedImages, dockerHubRegistry)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process addditional image %s", additionalImage)
		}
		newImages = append(newImages, newImage...)
	}

	return newImages, nil
}

func GetPrivateImages(baseDir string, kotsKindsImages []string, checkedImages map[string]types.ImageInfo, allPrivate bool, dockerHubRegistry dockerregistrytypes.RegistryOptions, parentHelmChartPath string, useHelmInstall map[string]bool) ([]string, []k8sdoc.K8sDoc, error) {
	uniqueImages := make(map[string]bool)
	objectsWithImages := make([]k8sdoc.K8sDoc, 0) // all objects where images are referenced from

	var mtx sync.Mutex
	const concurrencyLimit = 10
	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(concurrencyLimit)

	isPrivateImage := func(image string) (bool, error) {
		if allPrivate {
			return true, nil
		}

		mtx.Lock()
		checkedImage, ok := checkedImages[image]
		mtx.Unlock()
		if ok {
			return checkedImage.IsPrivate, nil
		}

		p, err := IsPrivateImage(image, dockerHubRegistry)
		if err != nil {
			return false, err
		}
		return p, nil
	}

	for _, image := range kotsKindsImages {
		func(image string) {
			g.Go(func() error {
				isPrivate, err := isPrivateImage(image)
				if err != nil {
					return errors.Wrapf(err, "failed to check if kotskinds image %s is private", image)
				}
				mtx.Lock()
				checkedImages[image] = types.ImageInfo{IsPrivate: isPrivate}
				mtx.Unlock()
				return nil
			})
		}(image)
	}

	err := filepath.Walk(baseDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				chartName, err := filepath.Rel(baseDir, path)
				if err != nil {
					logger.Debugf("Failed to remove prefix from %s: %v", path, err)
					return nil
				}
				chartName = filepath.Join(parentHelmChartPath, chartName)
				if useHelmInstall[chartName] {
					return filepath.SkipDir
				}
				return nil
			}

			contents, err := ioutil.ReadFile(path)
			if err != nil {
				logger.Debugf("Failed to read file %s: %v", path, err)
				return err
			}

			return listImagesInFile(contents, func(images []string, doc k8sdoc.K8sDoc) error {
				for _, image := range images {
					func(image string) {
						g.Go(func() error {
							isPrivate, err := isPrivateImage(image)
							if err != nil {
								return errors.Wrapf(err, "failed to check if image %s is private", image)
							}
							mtx.Lock()
							checkedImages[image] = types.ImageInfo{IsPrivate: isPrivate}
							if isPrivate {
								uniqueImages[image] = true
							}
							mtx.Unlock()
							return nil
						})
					}(image)
				}

				if len(images) > 0 {
					objectsWithImages = append(objectsWithImages, doc)
				}
				return nil
			})
		})

	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to walk upstream dir")
	}

	if err := g.Wait(); err != nil {
		return nil, nil, errors.Wrap(err, "failed to wait for image checks")
	}

	result := make([]string, 0, len(uniqueImages))
	for i := range uniqueImages {
		result = append(result, i)
	}
	sort.Strings(result) // sort the images to get an ordered and reproducible output for easier testing

	return result, objectsWithImages, nil
}

func rewriteImagesInFileBetweenRegistries(srcRegistry, destRegistry dockerregistrytypes.RegistryOptions, appSlug string, log *logger.CLILogger, reportWriter io.Writer, fileData []byte, copyImages, allImagesPrivate bool, checkedImages map[string]types.ImageInfo, savedImages map[string]bool, dockerHubRegistry dockerregistrytypes.RegistryOptions) ([]kustomizeimage.Image, error) {
	newImages := []kustomizeimage.Image{}

	err := listImagesInFile(fileData, func(images []string, doc k8sdoc.K8sDoc) error {
		for _, image := range images {
			if _, saved := savedImages[image]; saved {
				continue
			}

			if copyImages {
				log.ChildActionWithSpinner("Transferring image %s", image)
			} else {
				log.ChildActionWithSpinner("Found image %s", image)
			}

			newImage, err := rewriteOneImage(srcRegistry, destRegistry, image, appSlug, reportWriter, log, copyImages, allImagesPrivate, checkedImages, dockerHubRegistry)
			if err != nil {
				log.FinishChildSpinner()
				return errors.Wrapf(err, "failed to transfer image %s", image)
			}

			if newImage != nil {
				newImages = append(newImages, newImage...)
			}

			log.FinishChildSpinner()
			savedImages[image] = true
		}

		return nil
	})

	return newImages, err
}

type processImagesFunc func([]string, k8sdoc.K8sDoc) error

func listImagesInFile(contents []byte, handler processImagesFunc) error {
	yamlDocs := util.YAMLBytesToSingleDocs(contents)
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

func rewriteOneImage(srcRegistry, destRegistry dockerregistrytypes.RegistryOptions, image string, appSlug string, reportWriter io.Writer, log *logger.CLILogger, copyImages, allImagesPrivate bool, checkedImages map[string]types.ImageInfo, dockerHubRegistry dockerregistrytypes.RegistryOptions) ([]kustomizeimage.Image, error) {
	sourceCtx := &containerstypes.SystemContext{DockerDisableV1Ping: true}

	// allow pulling images from http/invalid https docker repos
	// intended for development only, _THIS MAKES THINGS INSECURE_
	if os.Getenv("KOTSADM_INSECURE_SRCREGISTRY") == "true" {
		sourceCtx.DockerInsecureSkipTLSVerify = containerstypes.OptionalBoolTrue
	}

	isPrivate := allImagesPrivate // rewrite all images with airgap
	if i, ok := checkedImages[image]; ok {
		isPrivate = i.IsPrivate
	} else {
		if !allImagesPrivate {
			p, err := IsPrivateImage(image, dockerHubRegistry)
			if err != nil {
				return nil, errors.Wrap(err, "failed to check if image is private")
			}
			isPrivate = p
		}
		checkedImages[image] = types.ImageInfo{
			IsPrivate: isPrivate,
		}
	}

	// TODO: This reaches out to internet in airgap installs. It shouldn't.
	sourceImage := image
	if isPrivate {
		sourceCtx.DockerAuthConfig = &containerstypes.DockerAuthConfig{
			Username: srcRegistry.Username,
			Password: srcRegistry.Password,
		}
		rewritten, err := RewritePrivateImage(srcRegistry, image, appSlug)
		if err != nil {
			return nil, errors.Wrap(err, "failed to rewrite private image")
		}
		sourceImage = rewritten
	}

	// normalize image to make sure only either a digest or a tag exist but not both
	parsedSrc, err := reference.ParseDockerRef(sourceImage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to normalize source image")
	}
	sourceImage = parsedSrc.String()

	srcRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s", sourceImage))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse source image name %s", sourceImage)
	}

	destImage, err := DestImage(destRegistry, image)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get destination image")
	}
	destStr := fmt.Sprintf("docker://%s", destImage)
	destRef, err := alltransports.ParseImageName(destStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse dest image name %s", destStr)
	}

	destCtx := &containerstypes.SystemContext{
		DockerInsecureSkipTLSVerify: containerstypes.OptionalBoolTrue,
		DockerDisableV1Ping:         true,
	}

	username, password := destRegistry.Username, destRegistry.Password
	registryHost := reference.Domain(destRef.DockerReference())

	if registry.IsECREndpoint(registryHost) && username != "AWS" {
		login, err := registry.GetECRLogin(registryHost, username, password)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get ECR login")
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

	if !copyImages {
		return kustomizeImage(destRegistry, image)
	}

	imageListSelection := copy.CopySystemImage
	if _, ok := parsedSrc.(reference.Canonical); ok {
		// this could be a multi-arch image, copy all architectures so that the digests match.
		imageListSelection = copy.CopyAllImages
	}

	_, err = CopyImageWithGC(context.Background(), destRef, srcRef, &copy.Options{
		RemoveSignatures:      true,
		SignBy:                "",
		ReportWriter:          reportWriter,
		SourceCtx:             sourceCtx,
		DestinationCtx:        destCtx,
		ForceManifestMIMEType: "",
		ImageListSelection:    imageListSelection,
	})
	if err != nil {
		log.Info("failed to copy image directly with error %q, attempting fallback transfer method", err.Error())
		// direct image copy failed
		// attempt to download image to a temp directory, and then upload it from there
		// this implicitly causes an image format conversion

		// make a temp directory
		tempDir, err := ioutil.TempDir("", "temp-image-pull")
		if err != nil {
			return nil, errors.Wrapf(err, "temp directory %s not created", tempDir)
		}
		defer os.RemoveAll(tempDir)

		destPath := path.Join(tempDir, "temp-archive-image")
		destStr := fmt.Sprintf("%s:%s", dockertypes.FormatDockerArchive, destPath)
		localRef, err := alltransports.ParseImageName(destStr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse local image name: %s", destStr)
		}

		// copy image from remote to local
		_, err = CopyImageWithGC(context.Background(), localRef, srcRef, &copy.Options{
			RemoveSignatures:      true,
			SignBy:                "",
			ReportWriter:          reportWriter,
			SourceCtx:             sourceCtx,
			DestinationCtx:        nil,
			ForceManifestMIMEType: "",
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to download image")
		}

		// copy image from local to remote
		_, err = CopyImageWithGC(context.Background(), destRef, localRef, &copy.Options{
			RemoveSignatures:      true,
			SignBy:                "",
			ReportWriter:          reportWriter,
			SourceCtx:             nil,
			DestinationCtx:        destCtx,
			ForceManifestMIMEType: "",
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to push image")
		}
	}

	return kustomizeImage(destRegistry, image)
}

func CopyImage(opts types.CopyImageOptions) error {
	srcCtx := &containerstypes.SystemContext{}
	destCtx := &containerstypes.SystemContext{}

	if opts.SkipSrcTLSVerify {
		srcCtx = &containerstypes.SystemContext{
			DockerInsecureSkipTLSVerify: containerstypes.OptionalBoolTrue,
			DockerDisableV1Ping:         true,
		}
	}

	if opts.SkipDestTLSVerify {
		destCtx = &containerstypes.SystemContext{
			DockerInsecureSkipTLSVerify: containerstypes.OptionalBoolTrue,
			DockerDisableV1Ping:         true,
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

		logger.Debugf("Marking image '%s' as private because: %v", image, err.Error())

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

type ProcessImageOptions struct {
	AppSlug          string
	Namespace        string
	RewriteImages    bool
	RegistrySettings regsitrytypes.RegistrySettings
	CopyImages       bool
	RootDir          string
	IsAirgap         bool
	AirgapRoot       string
	AirgapBundle     string
	PushImages       bool
	CreateAppDir     bool
	ReportWriter     io.Writer
}

// RewriteBaseImages Will rewrite images found in base and copy them (if necessary) to the configured registry.
func RewriteBaseImages(options ProcessImageOptions, baseDir string, kotsKinds *kotsutil.KotsKinds, license *kotsv1beta1.License, dockerHubRegistryCreds registry.Credentials, log *logger.CLILogger) (*RewriteImagesResult, error) {
	replicatedRegistryInfo := registry.GetRegistryProxyInfo(license, &kotsKinds.Installation, &kotsKinds.KotsApplication)

	rewriteImageOptions := RewriteImagesBetweenRegistriesOptions{
		BaseDir: baseDir,
		Log:     log,
		SourceRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:      replicatedRegistryInfo.Registry,
			ProxyEndpoint: replicatedRegistryInfo.Proxy,
		},
		DockerHubRegistry: dockerregistrytypes.RegistryOptions{
			Username: dockerHubRegistryCreds.Username,
			Password: dockerHubRegistryCreds.Password,
		},
		DestRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:  options.RegistrySettings.Hostname,
			Namespace: options.RegistrySettings.Namespace,
			Username:  options.RegistrySettings.Username,
			Password:  options.RegistrySettings.Password,
		},
		ReportWriter: options.ReportWriter,
		KotsKinds:    kotsKinds,
		IsAirgap:     options.IsAirgap,
		CopyImages:   options.CopyImages,
	}
	if license != nil {
		rewriteImageOptions.AppSlug = license.Spec.AppSlug
		rewriteImageOptions.SourceRegistry.Username = license.Spec.LicenseID
		rewriteImageOptions.SourceRegistry.Password = license.Spec.LicenseID
	}

	rewriteResult, err := RewriteImagesBetweenRegistries(rewriteImageOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to rewrite images")
	}

	return rewriteResult, nil
}

type RewriteImagesBetweenRegistriesOptions struct {
	BaseDir           string
	AppSlug           string
	SourceRegistry    dockerregistrytypes.RegistryOptions
	DestRegistry      dockerregistrytypes.RegistryOptions
	DockerHubRegistry dockerregistrytypes.RegistryOptions
	CopyImages        bool
	IsAirgap          bool
	Log               *logger.CLILogger
	ReportWriter      io.Writer
	KotsKinds         *kotsutil.KotsKinds
}

type RewriteImagesResult struct {
	Images        []kustomizeimage.Image          // images to be rewritten
	CheckedImages []kotsv1beta1.InstallationImage // all images found in the installation
}

func RewriteImagesBetweenRegistries(options RewriteImagesBetweenRegistriesOptions) (*RewriteImagesResult, error) {
	allImagesPrivate := options.IsAirgap
	additionalImages := make([]string, 0)
	checkedImages := make(map[string]types.ImageInfo)

	if options.KotsKinds != nil {
		additionalImages = kotsutil.GetImagesFromKotsKinds(options.KotsKinds)
		checkedImages = makeImageInfoMap(options.KotsKinds.Installation.Spec.KnownImages)
		if options.KotsKinds.KotsApplication.Spec.ProxyPublicImages {
			allImagesPrivate = true
		}
	}

	newImages, err := RewriteImages(options.SourceRegistry, options.DestRegistry, options.AppSlug, options.Log, options.ReportWriter, options.BaseDir, additionalImages, options.CopyImages, allImagesPrivate, checkedImages, options.DockerHubRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save images")
	}

	return &RewriteImagesResult{
		Images:        newImages,
		CheckedImages: makeInstallationImages(checkedImages),
	}, nil
}

func makeImageInfoMap(images []kotsv1beta1.InstallationImage) map[string]types.ImageInfo {
	result := make(map[string]types.ImageInfo)
	for _, i := range images {
		result[i.Image] = types.ImageInfo{
			IsPrivate: i.IsPrivate,
		}
	}
	return result
}

func makeInstallationImages(images map[string]types.ImageInfo) []kotsv1beta1.InstallationImage {
	result := make([]kotsv1beta1.InstallationImage, 0)
	for image, info := range images {
		result = append(result, kotsv1beta1.InstallationImage{
			Image:     image,
			IsPrivate: info.IsPrivate,
		})
	}
	return result
}
