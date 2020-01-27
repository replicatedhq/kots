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

	"github.com/containers/image/copy"
	imagedocker "github.com/containers/image/docker"
	dockerref "github.com/containers/image/docker/reference"
	"github.com/containers/image/signature"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/errcode"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/logger"
	"gopkg.in/yaml.v2"
	kustomizeimage "sigs.k8s.io/kustomize/v3/pkg/image"
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

func CopyImages(srcRegistry, destRegistry registry.RegistryOptions, appSlug string, log *logger.Logger, reportWriter io.Writer, upstreamDir string, dryRun bool) ([]kustomizeimage.Image, error) {
	savedImages := make(map[string]bool)
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

			newImagesSubset, err := copyImagesBetweenRegistries(srcRegistry, destRegistry, appSlug, log, reportWriter, contents, dryRun, savedImages)
			if err != nil {
				return errors.Wrapf(err, "failed to copy images mentioned in %s", path)
			}

			newImages = append(newImages, newImagesSubset...)
			return nil
		})

	if err != nil {
		return nil, errors.Wrap(err, "failed to walk upstream dir")
	}

	return newImages, nil
}

func GetPrivateImages(upstreamDir string) ([]string, []*k8sdoc.Doc, error) {
	checkedImages := make(map[string]bool)
	uniqueImages := make(map[string]bool)

	objects := make([]*k8sdoc.Doc, 0) // all objects where images are referenced from

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

			return listImagesInFile(contents, func(images []string, doc *k8sdoc.Doc) error {
				numPrivateImages := 0
				for _, image := range images {
					isPrivate := false
					if p, ok := checkedImages[image]; ok {
						isPrivate = p
					} else {
						p, err := isPrivateImage(image)
						if err != nil {
							return errors.Wrap(err, "failed to check if image is private")
						}
						isPrivate = p
						checkedImages[image] = p
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

func GetObjectsWithImages(upstreamDir string) ([]*k8sdoc.Doc, error) {
	objects := make([]*k8sdoc.Doc, 0)

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

			return listImagesInFile(contents, func(images []string, doc *k8sdoc.Doc) error {
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

func copyImagesBetweenRegistries(srcRegistry, destRegistry registry.RegistryOptions, appSlug string, log *logger.Logger, reportWriter io.Writer, fileData []byte, dryRun bool, savedImages map[string]bool) ([]kustomizeimage.Image, error) {
	newImages := []kustomizeimage.Image{}
	err := listImagesInFile(fileData, func(images []string, doc *k8sdoc.Doc) error {
		for _, image := range images {
			if _, saved := savedImages[image]; saved {
				continue
			}

			log.ChildActionWithSpinner("Transferring image %s", image)
			newImage, err := copyOneImage(srcRegistry, destRegistry, image, appSlug, reportWriter, log, dryRun)
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

type processImagesFunc func([]string, *k8sdoc.Doc) error

func listImagesInFile(contents []byte, handler processImagesFunc) error {
	yamlDocs := bytes.Split(contents, []byte("\n---\n"))
	for _, yamlDoc := range yamlDocs {
		parsed := &k8sdoc.Doc{}
		if err := yaml.Unmarshal(yamlDoc, parsed); err != nil {
			continue
		}

		images := make([]string, 0)
		for _, container := range parsed.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}

		for _, container := range parsed.Spec.Template.Spec.InitContainers {
			images = append(images, container.Image)
		}

		if err := handler(images, parsed); err != nil {
			return err
		}
	}

	return nil
}

func copyOneImage(srcRegistry, destRegistry registry.RegistryOptions, image string, appSlug string, reportWriter io.Writer, log *logger.Logger, dryRun bool) ([]kustomizeimage.Image, error) {
	policy, err := signature.NewPolicyFromBytes(imagePolicy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read default policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create policy")
	}

	sourceCtx := &types.SystemContext{}

	// allow pulling images from http/invalid https docker repos
	// intended for development only, _THIS MAKES THINGS INSECURE_
	if os.Getenv("KOTSADM_INSECURE_SRCREGISTRY") == "true" {
		sourceCtx.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	}

	isPrivate, err := isPrivateImage(image)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if image is private")
	}

	sourceImage := image
	if isPrivate {
		sourceCtx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: srcRegistry.Username,
			Password: srcRegistry.Password,
		}
		rewritten, err := rewritePrivateImage(srcRegistry, image, appSlug)
		if err != nil {
			return nil, errors.Wrap(err, "failed to rewrite private image")
		}

		sourceImage = rewritten
	}
	srcRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s", sourceImage))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse source image name %s", sourceImage)
	}

	destCtx := &types.SystemContext{
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
	}
	destCtx.DockerAuthConfig = &types.DockerAuthConfig{
		Username: destRegistry.Username,
		Password: destRegistry.Password,
	}

	destRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s", DestRef(destRegistry, image)))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse dest image name %s", DestRef(destRegistry, image))
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

func imageRefImage(image string) (*ImageRef, error) {
	ref := &ImageRef{}

	// named, err := reference.ParseNormalizedNamed(image)
	parsed, err := reference.ParseAnyReference(image)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse image name %q", image)
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
		return errors.Wrapf(err, "failed to parse dest image name: %s", destStr)
	}

	destCtx := &types.SystemContext{
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
	}

	if auth.Username != "" && auth.Password != "" {
		registryHost := reference.Domain(destRef.DockerReference())
		if registry.IsECREndpoint(registryHost) {
			login, err := registry.GetECRLogin(registryHost, auth.Username, auth.Password)
			if err != nil {
				return errors.Wrap(err, "failed to get ECR login")
			}
			auth.Username = login.Username
			auth.Password = login.Password
		}

		destCtx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: auth.Username,
			Password: auth.Password,
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

func isPrivateImage(image string) (bool, error) {
	// ParseReference requires the // prefix
	ref, err := imagedocker.ParseReference(fmt.Sprintf("//%s", image))
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse image ref:%s", image)
	}

	sysCtx := types.SystemContext{}

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

func rewritePrivateImage(srcRegistry registry.RegistryOptions, image string, appSlug string) (string, error) {
	ref, err := imagedocker.ParseReference(fmt.Sprintf("//%s", image))
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse image ref:%s", image)
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

	if err == imagedocker.ErrUnauthorizedForCredentials {
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
