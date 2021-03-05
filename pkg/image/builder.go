package image

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	imagedocker "github.com/containers/image/v5/docker"
	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/errcode"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	containerregistryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	cranetarball "github.com/google/go-containerregistry/pkg/v1/tarball"
	containerregistrytypes "github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/logger"
	kustomizeimage "sigs.k8s.io/kustomize/api/types"
)

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

func CopyImages(srcRegistry, destRegistry registry.RegistryOptions, appSlug string, log *logger.CLILogger, reportWriter io.Writer, upstreamDir string, additionalImages []string, dryRun, allImagesPrivate bool, checkedImages map[string]ImageInfo) ([]kustomizeimage.Image, error) {
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

func copyImageBetweenRegistries(srcRegistry, destRegistry registry.RegistryOptions, appSlug string, log *logger.CLILogger, reportWriter io.Writer, imageName string, dryRun, allImagesPrivate bool, checkedImages map[string]ImageInfo) ([]kustomizeimage.Image, error) {
	newImage, err := copyOneImage(srcRegistry, destRegistry, imageName, appSlug, reportWriter, log, dryRun, allImagesPrivate, checkedImages)
	if err != nil {
		log.FinishChildSpinner()
		return nil, errors.Wrapf(err, "failed to transfer image %s", imageName)
	}

	return newImage, nil
}

func copyImagesInFileBetweenRegistries(srcRegistry, destRegistry registry.RegistryOptions, appSlug string, log *logger.CLILogger, reportWriter io.Writer, fileData []byte, dryRun, allImagesPrivate bool, checkedImages map[string]ImageInfo, alreadyPushedImagesFromOtherFiles []kustomizeimage.Image) ([]kustomizeimage.Image, error) {
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

type Options struct {
	SrcRemoteOpts []remote.Option
	DstRemoteOpts []remote.Option
}

func DefaultRemoteOpts() *Options {
	platform := containerregistryv1.Platform{
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
	}

	opts := &Options{
		SrcRemoteOpts: []remote.Option{
			remote.WithPlatform(platform),
		},
		DstRemoteOpts: []remote.Option{},
	}

	return opts
}

func copyOneImage(srcRegistry, destRegistry registry.RegistryOptions, image string, appSlug string, reportWriter io.Writer, log *logger.CLILogger, dryRun, allImagesPrivate bool, checkedImages map[string]ImageInfo) ([]kustomizeimage.Image, error) {
	sourceCtx := &types.SystemContext{DockerDisableV1Ping: true}

	opts := DefaultRemoteOpts()

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

		authConfig := authn.AuthConfig{
			Username: srcRegistry.Username,
			Password: srcRegistry.Password,
		}
		opts.SrcRemoteOpts = append(opts.SrcRemoteOpts, remote.WithAuth(authn.FromConfig(authConfig)))

		sourceImage = rewritten
	}

	destImage := DestRef(destRegistry, image)
	destStr := fmt.Sprintf("docker://%s", destImage)
	destRef, err := alltransports.ParseImageName(destStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse dest image name %s", destStr)
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

		authConfig := authn.AuthConfig{
			Username: username,
			Password: password,
		}
		opts.DstRemoteOpts = append(opts.DstRemoteOpts, remote.WithAuth(authn.FromConfig(authConfig)))
	}

	if dryRun {
		return kustomizeImage(destRegistry, image)
	}

	err = CopyImageWithGC(sourceImage, destImage, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to copy image")
	}

	return kustomizeImage(destRegistry, image)
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
	destStr := fmt.Sprintf("docker://%s:%s", name, tag)
	destRef, err := alltransports.ParseImageName(destStr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse dest image name %s", destStr)
	}

	craneOptions := []crane.Option{
		crane.Insecure,
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

		authConfig := authn.AuthConfig{
			Username: username,
			Password: password,
		}
		craneOptions = append(craneOptions, crane.WithAuth(authn.FromConfig(authConfig)))
	}

	imageReader, err := RegistryImageFromReader(path)
	if err != nil {
		return errors.Wrap(err, "failed to create image reader 2")
	}

	dstImage := fmt.Sprintf("%s:%s", name, tag)
	err = PushImageFromStream(imageReader, dstImage, craneOptions)
	if err != nil {
		return errors.Wrap(err, "failed to copy image")
	}
	return nil
}

func IsPrivateImage(image string) (bool, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
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

		if strings.Contains(err.Error(), "EOF") {
			lastErr = err
			time.Sleep(1 * time.Second)
			continue
		}

		if !isUnauthorized(err) {
			return false, errors.Wrapf(err, "failed to create image from ref:%s", image)
		}

		return true, nil
	}

	return false, errors.Wrap(lastErr, "failed to retry")
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

func CopyImageWithGC(src string, dst string, opts *Options) error {
	srcRef, err := name.ParseReference(src)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %v", src, err)
	}

	dstRef, err := name.ParseReference(dst)
	if err != nil {
		return fmt.Errorf("parsing reference for %q: %v", dst, err)
	}

	desc, err := remote.Get(srcRef, opts.SrcRemoteOpts...)
	if err != nil {
		return fmt.Errorf("fetching %q: %v", src, err)
	}

	// copying an image increases allocated memory, which can push the pod to cross the memory limit when copying multiple images in a row.
	runGC := false
	defer func() {
		if runGC {
			runtime.GC()
		}
	}()

	if desc.MediaType == containerregistrytypes.DockerManifestSchema1 || desc.MediaType == containerregistrytypes.DockerManifestSchema1Signed {
		// TODO: "legacy" is an internal package and we can't import it
		// err = legacy.CopySchema1(desc, srcRef, dstRef, srcAuth, dstAuth)
		return errors.Errorf("unsupported media type: %s", desc.MediaType)
	}

	img, err := desc.Image()
	if err != nil {
		return errors.Wrap(err, "failed to read image")
	}

	err = remote.Write(dstRef, img, opts.DstRemoteOpts...)
	if err != nil {
		return errors.Wrap(err, "failed to write image")
	}

	runGC = true
	return nil
}

type airgapImageOpener struct {
	fileReader io.ReadCloser
	gzipReader io.ReadCloser
}

func (r airgapImageOpener) Read(p []byte) (int, error) {
	if r.gzipReader != nil {
		return r.gzipReader.Read(p)
	}
	return r.fileReader.Read(p)
}

func (r airgapImageOpener) Close() error {
	if r.gzipReader != nil {
		r.gzipReader.Close()
	}
	return r.fileReader.Close()
}

func RegistryImageFromReader(path string) (containerregistryv1.Image, error) {
	openerFunc := func() (io.ReadCloser, error) {
		fileReader, err := os.Open(path)
		if err != nil {
			return nil, errors.Wrap(err, "faile to open gzip file")
		}

		r := &airgapImageOpener{
			fileReader: fileReader,
		}

		gzipReader, err := gzip.NewReader(fileReader)
		if err == nil {
			r.gzipReader = gzipReader
			return r, nil
		}

		_, err = fileReader.Seek(0, 0)
		if err != nil {
			fileReader.Close()
			return nil, errors.Wrap(err, "failed to seek file")
		}

		return r, nil
	}
	return cranetarball.Image(openerFunc, nil)
}

func PushImageFromStream(image containerregistryv1.Image, dst string, opt []crane.Option) error {
	err := crane.Push(image, dst, opt...)

	// copying an image increases allocated memory, which can push the pod to cross the memory limit when copying multiple images in a row.
	runtime.GC()

	return err
}
