package image

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/copy"
	imagedocker "github.com/containers/image/docker"
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

func SaveImages(log *logger.Logger, imagesDir string, upstreamDir string) error {
	savedImages := make(map[string]bool)

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

			err = saveImagesFromFile(log, imagesDir, contents, savedImages)
			if err != nil {
				return errors.Wrap(err, "failed to extract images")
			}

			return nil
		})

	if err != nil {
		return errors.Wrap(err, "failed to walk upstream dir")
	}

	return nil
}

func GetPrivateImages(upstreamDir string) ([]string, []*k8sdoc.Doc, error) {
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
					isPrivate, err := isPrivateImage(image)
					if err != nil {
						return errors.Wrap(err, "failed to check if image is private")
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

func saveImagesFromFile(log *logger.Logger, imagesDir string, fileData []byte, savedImages map[string]bool) error {
	err := listImagesInFile(fileData, func(images []string, doc *k8sdoc.Doc) error {
		for _, image := range images {
			if _, saved := savedImages[image]; saved {
				continue
			}

			log.ChildActionWithSpinner("Pulling image %s", image)
			err := saveOneImage(imagesDir, image)
			if err != nil {
				log.FinishChildSpinner()
				return errors.Wrap(err, "failed to save image")
			}

			log.FinishChildSpinner()
			savedImages[image] = true
		}

		return nil
	})

	return err
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

		if err := handler(images, parsed); err != nil {
			return err
		}
	}

	return nil
}

func saveOneImage(imagesDir string, image string) error {
	imageRef, err := imageRefImage(image)
	if err != nil {
		return errors.Wrap(err, "failed to parse image ref")
	}

	imageFormat := "docker-archive"
	pathInBundle := imageRef.pathInBundle(imageFormat)
	archiveName := filepath.Join(imagesDir, pathInBundle)
	destDir := filepath.Dir(archiveName)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create destination dir")
	}

	policy, err := signature.NewPolicyFromBytes(imagePolicy)
	if err != nil {
		return errors.Wrap(err, "failed to read default policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return errors.Wrap(err, "failed to create policy")
	}

	srcRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s", image))
	if err != nil {
		return errors.Wrap(err, "failed to parse source image name")
	}

	destStr := fmt.Sprintf("%s:%s", imageFormat, archiveName)
	destRef, err := alltransports.ParseImageName(destStr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse local image name: %s", destStr)
	}

	_, err = copy.Image(context.Background(), policyContext, destRef, srcRef, &copy.Options{
		RemoveSignatures:      true,
		SignBy:                "",
		ReportWriter:          nil,
		SourceCtx:             nil,
		DestinationCtx:        nil,
		ForceManifestMIMEType: "",
	})
	if err != nil {
		return errors.Wrap(err, "failed to copy image")
	}

	return nil
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

	remoteImage, err := ref.NewImage(context.Background(), nil)
	if err == nil {
		remoteImage.Close()
		return false, nil
	}

	if !isUnauthorized(err) {
		return false, errors.Wrapf(err, "failed to create image from ref:%s", image)
	}

	return true, nil
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
