package image

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/containers/image/v5/copy"
	imagedocker "github.com/containers/image/v5/docker"
	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/errcode"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/logger"
	kustomizeimage "sigs.k8s.io/kustomize/api/types"
)

var imagePolicy = []byte(`{
  "default": [{"type": "insecureAcceptAnything"}]
}`)

type ImageRef struct {
	Domain string
	Name   string
	Tag    string
	Digest string
}

type RegistryAuth struct {
	Username string
	Password string
}

type ImageInfo struct {
	IsPrivate bool
}

func CopyImages(srcRegistry, destRegistry registry.RegistryOptions, appSlug string, log *logger.Logger, reportWriter io.Writer, upstreamDir string, additionalImages []string, dryRun, allImagesPrivate bool, checkedImages map[string]ImageInfo) ([]kustomizeimage.Image, error) {
	newImages := []kustomizeimage.Image{}

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

			newImagesSubset, err := copyImagesInFileBetweenRegistries(srcRegistry, destRegistry, appSlug, log, reportWriter, contents, dryRun, allImagesPrivate, checkedImages, newImages)
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
		newImagesSubset, err := copyImageBetweenRegistries(srcRegistry, destRegistry, appSlug, log, reportWriter, additionalImage, dryRun, allImagesPrivate, checkedImages)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to addditional image: %s", additionalImage)
		}
		newImages = append(newImages, newImagesSubset...)
	}

	return newImages, nil
}

func GetPrivateImages(upstreamDir string, checkedImages map[string]ImageInfo, allPrivate bool) ([]string, []k8sdoc.K8sDoc, error) {
	uniqueImages := make(map[string]bool)

	objects := make([]k8sdoc.K8sDoc, 0) // all objects where images are referenced from

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

			return listImagesInFile(contents, func(images []string, doc k8sdoc.K8sDoc) error {
				numPrivateImages := 0
				for idx, image := range images {
					if allPrivate {
						checkedImages[image] = ImageInfo{
							IsPrivate: true,
						}
						numPrivateImages = numPrivateImages + 1
						uniqueImages[image] = true
						continue
					}

					isPrivate := false
					if i, ok := checkedImages[image]; ok {
						isPrivate = i.IsPrivate
					} else {
						p, err := IsPrivateImage(image)
						if err != nil {
							return errors.Wrapf(err, "failed to check if image %d of %d in %q is private", idx+1, len(images), info.Name())
						}
						isPrivate = p
						checkedImages[image] = ImageInfo{
							IsPrivate: p,
						}
					}

					if !isPrivate {
						continue
					}
					numPrivateImages = numPrivateImages + 1
					uniqueImages[image] = true
				}

				if numPrivateImages == 0 {
					return nil
				}

				objects = append(objects, doc)
				return nil
			})
		})

	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to walk upstream dir")
	}

	result := make([]string, 0, len(uniqueImages))
	for i := range uniqueImages {
		result = append(result, i)
	}

	return result, objects, nil
}

func GetObjectsWithImages(upstreamDir string) ([]k8sdoc.K8sDoc, error) {
	objects := make([]k8sdoc.K8sDoc, 0)

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

			return listImagesInFile(contents, func(images []string, doc k8sdoc.K8sDoc) error {
				if len(images) > 0 {
					objects = append(objects, doc)
				}
				return nil
			})
		})

	if err != nil {
		return nil, errors.Wrap(err, "failed to walk upstream dir")
	}

	return objects, nil
}

func copyImageBetweenRegistries(srcRegistry, destRegistry registry.RegistryOptions, appSlug string, log *logger.Logger, reportWriter io.Writer, imageName string, dryRun, allImagesPrivate bool, checkedImages map[string]ImageInfo) ([]kustomizeimage.Image, error) {
	newImage, err := copyOneImage(srcRegistry, destRegistry, imageName, appSlug, reportWriter, log, dryRun, allImagesPrivate, checkedImages)
	if err != nil {
		log.FinishChildSpinner()
		return nil, errors.Wrapf(err, "failed to transfer image %s", imageName)
	}

	return newImage, nil
}

func copyImagesInFileBetweenRegistries(srcRegistry, destRegistry registry.RegistryOptions, appSlug string, log *logger.Logger, reportWriter io.Writer, fileData []byte, dryRun, allImagesPrivate bool, checkedImages map[string]ImageInfo, alreadyPushedImagesFromOtherFiles []kustomizeimage.Image) ([]kustomizeimage.Image, error) {
	savedImages := make(map[string]bool)
	newImages := []kustomizeimage.Image{}

	for _, image := range alreadyPushedImagesFromOtherFiles {
		savedImages[fmt.Sprintf("%s:%s", image.Name, image.NewTag)] = true
	}

	err := listImagesInFile(fileData, func(images []string, doc k8sdoc.K8sDoc) error {
		for _, image := range images {
			if _, saved := savedImages[image]; saved {
				continue
			}

			log.ChildActionWithSpinner("Transferring image %s", image)
			newImage, err := copyOneImage(srcRegistry, destRegistry, image, appSlug, reportWriter, log, dryRun, allImagesPrivate, checkedImages)
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
	yamlDocs := bytes.Split(contents, []byte("\n---\n"))
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

func copyOneImage(srcRegistry, destRegistry registry.RegistryOptions, image string, appSlug string, reportWriter io.Writer, log *logger.Logger, dryRun, allImagesPrivate bool, checkedImages map[string]ImageInfo) ([]kustomizeimage.Image, error) {
	policy, err := signature.NewPolicyFromBytes(imagePolicy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read default policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create policy")
	}

	sourceCtx := &types.SystemContext{DockerDisableV1Ping: true}

	// allow pulling images from http/invalid https docker repos
	// intended for development only, _THIS MAKES THINGS INSECURE_
	if os.Getenv("KOTSADM_INSECURE_SRCREGISTRY") == "true" {
		sourceCtx.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	}

	isPrivate := allImagesPrivate // rewrite all images with airgap
	if i, ok := checkedImages[image]; ok {
		isPrivate = i.IsPrivate
	} else {
		if !allImagesPrivate {
			p, err := IsPrivateImage(image)
			if err != nil {
				return nil, errors.Wrap(err, "failed to check if image is private")
			}
			isPrivate = p
		}
		checkedImages[image] = ImageInfo{
			IsPrivate: isPrivate,
		}
	}

	// TODO: This reaches out to internet in airgap installs.  It shouldn't.
	sourceImage := image
	if isPrivate {
		sourceCtx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: srcRegistry.Username,
			Password: srcRegistry.Password,
		}
		rewritten, err := RewritePrivateImage(srcRegistry, image, appSlug)
		if err != nil {
			return nil, errors.Wrap(err, "failed to rewrite private image")
		}

		sourceImage = rewritten
	}
	srcRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s", sourceImage))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse source image name %s", sourceImage)
	}

	destStr := fmt.Sprintf("docker://%s", DestRef(destRegistry, image))
	destRef, err := alltransports.ParseImageName(destStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse dest image name %s", destStr)
	}

	destCtx := &types.SystemContext{
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
		DockerDisableV1Ping:         true,
	}

	if destRegistry.Username != "" && destRegistry.Password != "" {
		username, password := destRegistry.Username, destRegistry.Password

		registryHost := reference.Domain(destRef.DockerReference())
		if registry.IsECREndpoint(registryHost) {
			login, err := registry.GetECRLogin(registryHost, username, password)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get ECR login")
			}
			username = login.Username
			password = login.Password
		}

		destCtx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: username,
			Password: password,
		}
	}

	if dryRun {
		return buildImageAlts(destRegistry, image)
	}

	_, err = copy.Image(context.Background(), policyContext, destRef, srcRef, &copy.Options{
		RemoveSignatures:      true,
		SignBy:                "",
		ReportWriter:          reportWriter,
		SourceCtx:             sourceCtx,
		DestinationCtx:        destCtx,
		ForceManifestMIMEType: "",
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
		destStr := fmt.Sprintf("docker-archive:%s", destPath)
		localRef, err := alltransports.ParseImageName(destStr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse local image name: %s", destStr)
		}

		// copy image from remote to local
		_, err = copy.Image(context.Background(), policyContext, localRef, srcRef, &copy.Options{
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
		_, err = copy.Image(context.Background(), policyContext, destRef, localRef, &copy.Options{
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

	return buildImageAlts(destRegistry, image)
}

func RefFromImage(image string) (*ImageRef, error) {
	ref := &ImageRef{}

	// named, err := reference.ParseNormalizedNamed(image)
	parsed, err := reference.ParseAnyReference(image)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse image ref %q", image)
	}

	if named, ok := parsed.(reference.Named); ok {
		ref.Domain = reference.Domain(named)
		ref.Name = named.Name()
	} else {
		return nil, errors.New(fmt.Sprintf("unsupported ref type: %T", parsed))
	}

	if tagged, ok := parsed.(reference.Tagged); ok {
		ref.Tag = tagged.Tag()
	} else if can, ok := parsed.(reference.Canonical); ok {
		ref.Digest = can.Digest().String()
	} else {
		ref.Tag = "latest"
	}

	return ref, nil
}

func (ref *ImageRef) pathInBundle(formatPrefix string) string {
	path := []string{formatPrefix, ref.Name}
	if ref.Tag != "" {
		path = append(path, ref.Tag)
	}
	if ref.Digest != "" {
		digestParts := strings.Split(ref.Digest, ":")
		path = append(path, digestParts...)
	}
	return filepath.Join(path...)
}

func (ref *ImageRef) NameBase() string {
	return path.Base(ref.Name)
}

func (ref *ImageRef) String() string {
	refStr := ref.Name
	if ref.Tag != "" {
		refStr = fmt.Sprintf("%s:%s", refStr, ref.Tag)
	} else if ref.Domain != "" {
		refStr = fmt.Sprintf("%s@%s", refStr, ref.Digest)
	}
	return refStr
}

func CopyFromFileToRegistry(path string, name string, tag string, digest string, auth RegistryAuth, reportWriter io.Writer) error {
	policy, err := signature.NewPolicyFromBytes(imagePolicy)
	if err != nil {
		return errors.Wrap(err, "failed to read default policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return errors.Wrap(err, "failed to create policy")
	}

	srcRef, err := alltransports.ParseImageName(fmt.Sprintf("docker-archive:%s", path))
	if err != nil {
		return errors.Wrap(err, "failed to parse src image name")
	}

	destStr := fmt.Sprintf("docker://%s:%s", name, tag)
	destRef, err := alltransports.ParseImageName(destStr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse dest image name %s", destStr)
	}

	destCtx := &types.SystemContext{
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
		DockerDisableV1Ping: true,
	}

	if auth.Username != "" && auth.Password != "" {
		username, password := auth.Username, auth.Password

		registryHost := reference.Domain(destRef.DockerReference())
		if registry.IsECREndpoint(registryHost) {
			login, err := registry.GetECRLogin(registryHost, username, password)
			if err != nil {
				return errors.Wrap(err, "failed to get ECR login")
			}
			username = login.Username
			password = login.Password
		}

		destCtx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: username,
			Password: password,
		}
	}

	_, err = copy.Image(context.Background(), policyContext, destRef, srcRef, &copy.Options{
		RemoveSignatures:      true,
		SignBy:                "",
		ReportWriter:          reportWriter,
		SourceCtx:             nil,
		DestinationCtx:        destCtx,
		ForceManifestMIMEType: "",
	})
	if err != nil {
		return errors.Wrap(err, "failed to copy image")
	}

	return nil
}

func IsPrivateImage(image string) (bool, error) {
	// ParseReference requires the // prefix
	ref, err := imagedocker.ParseReference(fmt.Sprintf("//%s", image))
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse image ref %q", image)
	}

	sysCtx := types.SystemContext{DockerDisableV1Ping: true}

	// allow pulling images from http/invalid https docker repos
	// intended for development only, _THIS MAKES THINGS INSECURE_
	if os.Getenv("KOTSADM_INSECURE_SRCREGISTRY") == "true" {
		sysCtx.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	}

	remoteImage, err := ref.NewImage(context.Background(), &sysCtx)
	if err == nil {
		remoteImage.Close()
		return false, nil
	}

	// manifest was downloaded, but no matching architecture found in manifest.
	// still, not a private image
	if strings.Contains(err.Error(), "no image found in manifest list for architecture") {
		return false, nil
	}

	if !isUnauthorized(err) {
		return false, errors.Wrapf(err, "failed to create image from ref:%s", image)
	}

	return true, nil
}

func RewritePrivateImage(srcRegistry registry.RegistryOptions, image string, appSlug string) (string, error) {
	ref, err := imagedocker.ParseReference(fmt.Sprintf("//%s", image))
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse image ref %q", image)
	}

	registryHost := dockerref.Domain(ref.DockerReference())
	if registryHost == srcRegistry.Endpoint {
		// replicated images are also private, but we don't rewrite those
		return image, nil
	}

	newImage := registry.MakeProxiedImageURL(srcRegistry.ProxyEndpoint, appSlug, image)
	if tagged, ok := ref.DockerReference().(dockerref.Tagged); ok {
		return newImage + ":" + tagged.Tag(), nil
	} else if can, ok := ref.DockerReference().(reference.Canonical); ok {
		return newImage + "@" + can.Digest().String(), nil
	}

	// no tag, so it will be "latest"
	return newImage, nil
}

func isUnauthorized(err error) bool {
	switch err := err.(type) {
	case errcode.Errors:
		for _, e := range err {
			if isUnauthorized(e) {
				return true
			}
		}
		return false
	case errcode.Error:
		return err.Code.Descriptor().HTTPStatusCode == http.StatusUnauthorized
	}

	if _, ok := err.(imagedocker.ErrUnauthorizedForCredentials); ok {
		return true
	}

	cause := errors.Cause(err)
	if cause, ok := cause.(error); ok {
		if cause == err {
			return false
		}
	}

	return isUnauthorized(cause)
}
