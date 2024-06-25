package imageutil

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/containers/image/v5/docker/reference"

	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/pkg/errors"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

// GetTag extracts the image tag from an image reference
func GetTag(imageRef string) (string, error) {
	ref, err := reference.ParseNormalizedNamed(imageRef)
	if err != nil {
		return "", err
	}
	if tagged, ok := ref.(reference.Tagged); ok {
		return tagged.Tag(), nil
	}
	return "", fmt.Errorf("image reference is not tagged")
}

func RewriteDockerArchiveImage(registry registrytypes.RegistryOptions, nameParts []string) (kustomizetypes.Image, error) {
	// imageNameParts looks like this:
	// ["quay.io", "someorg", "imagename", "imagetag"]
	// or
	// ["quay.io", "someorg", "imagename", "sha256", "<sha>"]
	// we want to discard everything upto "imagename" and replace that with local host and namespace

	image := kustomizetypes.Image{}

	if len(nameParts) < 3 {
		return image, errors.Errorf("not enough parts in image name: %v", nameParts)
	}

	newImageNameParts := []string{registry.Endpoint}
	if registry.Namespace != "" {
		newImageNameParts = append(newImageNameParts, registry.Namespace)
	}
	var originalName string
	if nameParts[len(nameParts)-2] == "sha256" {
		newImageNameParts = append(newImageNameParts, nameParts[len(nameParts)-3])
		originalName = path.Join(nameParts[:len(nameParts)-2]...)
		image.Digest = fmt.Sprintf("sha256:%s", nameParts[len(nameParts)-1])
	} else {
		newImageNameParts = append(newImageNameParts, nameParts[len(nameParts)-2])
		originalName = path.Join(nameParts[:len(nameParts)-1]...)
		image.NewTag = fmt.Sprintf("%s", nameParts[len(nameParts)-1])
	}

	image.Name = originalName
	image.NewName = path.Join(newImageNameParts...)

	return image, nil
}

func RewriteDockerRegistryImage(destRegistry registrytypes.RegistryOptions, srcImage string) (*kustomizetypes.Image, error) {
	parsedSrc, err := reference.ParseDockerRef(srcImage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to normalize source image")
	}

	destImage, err := DestImage(destRegistry, srcImage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get destination image")
	}

	rewrittenImage := kustomizetypes.Image{}
	rewrittenImage.Name = StripImageTagAndDigest(srcImage)
	rewrittenImage.NewName = StripImageTagAndDigest(destImage)

	if can, ok := parsedSrc.(reference.Canonical); ok {
		rewrittenImage.Digest = can.Digest().String()
	} else if tagged, ok := parsedSrc.(reference.Tagged); ok {
		rewrittenImage.NewTag = tagged.Tag()
	} else {
		rewrittenImage.NewTag = "latest"
	}

	return &rewrittenImage, nil
}

// DestImage returns the location to push the image to on the dest registry
func DestImage(destRegistry registrytypes.RegistryOptions, srcImage string) (string, error) {
	// parsing as a docker reference strips the tag if both a tag and a digest are used
	parsed, err := reference.ParseDockerRef(srcImage)
	if err != nil {
		return "", errors.Wrap(err, "failed to normalize source image")
	}
	srcImage = parsed.String()

	imageParts := strings.Split(srcImage, "/")
	lastPart := imageParts[len(imageParts)-1]

	return filepath.Join(destRegistry.Endpoint, destRegistry.Namespace, lastPart), nil
}

// DestECImage returns the location to push an embedded cluster image to on the dest registry
func DestECImage(destRegistry registrytypes.RegistryOptions, srcImage string) (string, error) {
	// parsing as a docker reference strips the tag if both a tag and a digest are used
	parsed, err := reference.ParseDockerRef(srcImage)
	if err != nil {
		return "", errors.Wrap(err, "failed to normalize source image")
	}
	srcImage = parsed.String()

	imageParts := strings.Split(srcImage, "/")
	lastPart := imageParts[len(imageParts)-1]

	return filepath.Join(destRegistry.Endpoint, destRegistry.Namespace, "embedded-cluster", lastPart), nil
}

// DestImageFromKustomizeImage returns the location to push the image to from a kustomize image type
func DestImageFromKustomizeImage(image kustomizetypes.Image) string {
	destImage := image.NewName

	if image.Digest != "" {
		destImage += "@"
		destImage += image.Digest
	} else if image.NewTag != "" {
		destImage += ":"
		destImage += image.NewTag
	}

	return destImage
}

func BuildImageAltNames(rewrittenImage kustomizetypes.Image) ([]kustomizetypes.Image, error) {
	// kustomize does string based comparison, so all of these are treated as different images:
	// docker.io/library/redis:latest
	// docker.io/redis:latest
	// redis:latest
	// redis
	// As a workaround we add all 4 to the list

	// similarly, docker.io/notlibrary/image:tag needs to be rewritten
	// as notlibrary/image:tag

	// if host is not docker.io, then only return the original image

	dockerRef, err := dockerref.ParseDockerRef(rewrittenImage.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "parse docker ref: %q", rewrittenImage.Name)
	}

	images := []kustomizetypes.Image{rewrittenImage}

	registryHost := dockerref.Domain(dockerRef)
	if registryHost != "docker.io" && !strings.HasSuffix(registryHost, ".docker.io") {
		return images, nil
	}

	nameParts := strings.Split(rewrittenImage.Name, "/")

	if len(nameParts) > 2 && nameParts[0] == "docker.io" && nameParts[1] == "library" {
		// This is a docker library image, 4 possible variations
		nameParts = nameParts[1:] // remove "docker.io"
		images = append(images, kustomizetypes.Image{
			Name:    path.Join(nameParts...),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
		nameParts = nameParts[1:] // remove "library"
		images = append(images, kustomizetypes.Image{
			Name:    path.Join(nameParts...),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
		nameParts = append([]string{"docker.io"}, nameParts...) // add "docker.io", without "library"
		images = append(images, kustomizetypes.Image{
			Name:    path.Join(nameParts...),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
	} else if len(nameParts) == 2 && nameParts[0] == "docker.io" {
		// This is a docker library image, 4 possible variations
		nameParts = nameParts[1:] // remove "docker.io"
		images = append(images, kustomizetypes.Image{
			Name:    path.Join(nameParts...),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
		images = append(images, kustomizetypes.Image{
			Name:    strings.Join(append([]string{"docker.io", "library"}, nameParts...), "/"), // add "docker.io/library"
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
		nameParts = append([]string{"library"}, nameParts...) // add "library"
		images = append(images, kustomizetypes.Image{
			Name:    path.Join(nameParts...),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
	} else if len(nameParts) > 2 && nameParts[0] == "docker.io" {
		// This is a docker non-library image, 2 possible variations
		nameParts = nameParts[1:] // remove "docker.io"
		images = append(images, kustomizetypes.Image{
			Name:    path.Join(nameParts...),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
	} else if len(nameParts) > 1 && nameParts[0] == "library" {
		// This is a docker library image, 4 possible variations
		nameParts = nameParts[1:] // remove "library"
		images = append(images, kustomizetypes.Image{
			Name:    path.Join(nameParts...),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
		images = append(images, kustomizetypes.Image{
			Name:    strings.Join(append([]string{"docker.io"}, nameParts...), "/"),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
		images = append(images, kustomizetypes.Image{
			Name:    strings.Join(append([]string{"docker.io", "library"}, nameParts...), "/"),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
	} else if len(nameParts) == 1 {
		// This is a docker library image, 4 possible variations
		images = append(images, kustomizetypes.Image{
			Name:    strings.Join([]string{"docker.io", "library", nameParts[0]}, "/"),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
		images = append(images, kustomizetypes.Image{
			Name:    strings.Join([]string{"library", nameParts[0]}, "/"),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
		images = append(images, kustomizetypes.Image{
			Name:    strings.Join([]string{"docker.io", nameParts[0]}, "/"),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
	} else if len(nameParts) == 2 {
		// This is a docker non-library image, 2 possible variations
		images = append(images, kustomizetypes.Image{
			Name:    strings.Join([]string{"docker.io", nameParts[0], nameParts[1]}, "/"),
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
	}

	return images, nil
}

// StripImageTagAndDigest removes the tag and digest from an image while preserving the original name.
// This can be helpful because parsing the image as a docker reference can modify the hostname (e.g. adds docker.io/library)
func StripImageTagAndDigest(image string) string {
	// grab last section of image name
	imageParts := strings.Split(image, "/")
	lastPart := imageParts[len(imageParts)-1]

	// strip tag (like 'img:tag') or digest (like 'img@sha256:sha')
	lastPart = strings.Split(lastPart, "@")[0]
	lastPart = strings.Split(lastPart, ":")[0]
	imageParts[len(imageParts)-1] = lastPart

	// rejoin parts of image name
	image = strings.Join(imageParts, "/")
	return image
}

func KustomizeImage(destRegistry registrytypes.RegistryOptions, image string) ([]kustomizetypes.Image, error) {
	rewrittenImage, err := RewriteDockerRegistryImage(destRegistry, image)
	if err != nil {
		return nil, errors.Wrap(err, "failed to rewrite image")
	}
	rewrittenImages, err := BuildImageAltNames(*rewrittenImage)
	if err != nil {
		return nil, errors.Wrap(err, "build image name alts")
	}
	return rewrittenImages, nil
}

// GetImageName returns the name of the image without the tag, digest, or registry
func GetImageName(image string) string {
	imageParts := strings.Split(image, "/")
	lastPart := imageParts[len(imageParts)-1]
	return StripImageTagAndDigest(lastPart)
}

// ChangeImageTag changes the tag of an image to the provided new tag
func ChangeImageTag(image string, newTag string) (string, error) {
	parsed, err := reference.ParseDockerRef(image)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse image")
	}

	if _, ok := parsed.(reference.Canonical); ok {
		// TODO: change tag for digested image that also has a tag
		return image, nil
	}

	if _, ok := parsed.(reference.Tagged); !ok {
		// image is not tagged, just append the tag
		return fmt.Sprintf("%s:%s", image, newTag), nil
	}

	imageParts := strings.Split(image, "/")
	lastPart := imageParts[len(imageParts)-1]
	lastPart = fmt.Sprintf("%s:%s", strings.Split(lastPart, ":")[0], newTag)
	imageParts[len(imageParts)-1] = lastPart

	return strings.Join(imageParts, "/"), nil
}

// A valid tag must be valid ASCII and can contain lowercase and uppercase letters, digits, underscores, periods, and hyphens.
// It can't start with a period or hyphen and must be no longer than 128 characters.
// ref: https://docs.docker.com/reference/cli/docker/image/tag/#description
func SanitizeTag(tag string) string {
	tag = strings.Join(dockerref.TagRegexp.FindAllString(tag, -1), "")
	if len(tag) > 128 {
		tag = tag[:128]
	}
	return tag
}

// A valid repo may contain lowercase letters, digits and separators.
// A separator is defined as a period, one or two underscores, or one or more hyphens.
// A component may not start or end with a separator.
// ref: https://docs.docker.com/reference/cli/docker/image/tag/#description
func SanitizeRepo(repo string) string {
	repo = strings.ToLower(repo)
	repo = strings.Join(dockerref.NameRegexp.FindAllString(repo, -1), "")
	return repo
}

type OCIArtifactPath struct {
	Name              string
	RegistryHost      string
	RegistryNamespace string
	Repository        string
	Tag               string
}

func (p *OCIArtifactPath) String() string {
	if p.RegistryNamespace == "" {
		return fmt.Sprintf("%s:%s", filepath.Join(p.RegistryHost, p.Repository), p.Tag)
	}
	return fmt.Sprintf("%s:%s", filepath.Join(p.RegistryHost, p.RegistryNamespace, p.Repository), p.Tag)
}

type EmbeddedClusterArtifactOCIPathOptions struct {
	RegistryHost      string
	RegistryNamespace string
	ChannelID         string
	UpdateCursor      string
	VersionLabel      string
}

// NewEmbeddedClusterOCIArtifactPath returns the OCI path for an embedded cluster artifact given
// the artifact filename and details about the configured registry and channel release.
func NewEmbeddedClusterOCIArtifactPath(filename string, opts EmbeddedClusterArtifactOCIPathOptions) *OCIArtifactPath {
	name := filepath.Base(filename)
	repository := filepath.Join("embedded-cluster", SanitizeRepo(name))
	tag := SanitizeTag(fmt.Sprintf("%s-%s-%s", opts.ChannelID, opts.UpdateCursor, opts.VersionLabel))
	return &OCIArtifactPath{
		Name:              name,
		RegistryHost:      opts.RegistryHost,
		RegistryNamespace: opts.RegistryNamespace,
		Repository:        repository,
		Tag:               tag,
	}
}
