package image

import (
	"fmt"
	"path"
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
	var originalName, ref, separator string
	if nameParts[len(nameParts)-2] == "sha256" {
		newImageNameParts = append(newImageNameParts, nameParts[len(nameParts)-3])
		originalName = path.Join(nameParts[:len(nameParts)-2]...)
		ref = fmt.Sprintf("sha256:%s", nameParts[len(nameParts)-1])
		separator = "@"
		image.Digest = ref
	} else {
		newImageNameParts = append(newImageNameParts, nameParts[len(nameParts)-2])
		originalName = path.Join(nameParts[:len(nameParts)-1]...)
		ref = fmt.Sprintf("%s", nameParts[len(nameParts)-1])
		separator = ":"
		image.NewTag = ref
	}

	image.Name = fmt.Sprintf("%s%s%s", originalName, separator, ref)
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
	parsedDest, err := reference.ParseDockerRef(destImage)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse destination image %s", destImage)
	}

	rewrittenImage := kustomizetypes.Image{}
	rewrittenImage.Name = srcImage
	rewrittenImage.NewName = reference.TrimNamed(parsedDest).Name()

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

	if destRegistry.Namespace == "" {
		return fmt.Sprintf("%s/%s", destRegistry.Endpoint, lastPart), nil
	}
	return fmt.Sprintf("%s/%s/%s", destRegistry.Endpoint, destRegistry.Namespace, lastPart), nil
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

// stripImageTag removes the tag or digest from an image
func stripImageTag(image string) string {
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

// destImageName returns the name of the image on the dest registry (without tag or digest)
func destImageName(registry registrytypes.RegistryOptions, srcImage string) string {
	imageParts := strings.Split(srcImage, "/")
	lastPart := imageParts[len(imageParts)-1]
	lastPart = stripImageTag(lastPart)

	if registry.Namespace == "" {
		return fmt.Sprintf("%s/%s", registry.Endpoint, lastPart)
	}
	return fmt.Sprintf("%s/%s/%s", registry.Endpoint, registry.Namespace, lastPart)
}

func kustomizeImage(destRegistry registrytypes.RegistryOptions, image string) ([]kustomizetypes.Image, error) {
	imgParts := strings.Split(image, "/")

	dockerRef, err := dockerref.ParseDockerRef(image)
	if err != nil {
		return nil, errors.Wrapf(err, "parse docker ref: %q", image)
	}

	registryHost := dockerref.Domain(dockerRef)
	if registryHost == "docker.io" || strings.HasSuffix(registryHost, ".docker.io") {
		if len(imgParts) == 1 {
			// this means the image is something like "redis", which refers to "docker.io/library/redis"
			imgParts = append([]string{"docker.io", "library"}, imgParts...)
		} else if len(imgParts) == 2 && imgParts[0] == "library" {
			// this means the image is something like "kotsadm/kotsadm", which refers to "docker.io/kotsadm/kotsadm"
			imgParts = append([]string{"docker.io"}, imgParts...)
		}
	}

	// if the last substring doesn't contain ':', it is untagged and needs 'latest' appended
	// otherwise, split it on '@' and then ':'
	if strings.Contains(imgParts[len(imgParts)-1], ":") {
		imgParts = append(imgParts[:len(imgParts)-1], strings.Split(imgParts[len(imgParts)-1], "@")...)
		imgParts = append(imgParts[:len(imgParts)-1], strings.Split(imgParts[len(imgParts)-1], ":")...)
	} else {
		imgParts = append(imgParts, "latest")
	}

	imageInfo, err := RewriteDockerArchiveImage(destRegistry, imgParts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to rewrite docker archive image")
	}

	newName := destImageName(destRegistry, image)
	imageWithoutTag := stripImageTag(image)

	kustomizedImage := kustomizetypes.Image{
		Name:    imageWithoutTag,
		NewName: newName,
		NewTag:  imageInfo.NewTag,
		Digest:  imageInfo.Digest,
	}
	images, err := BuildImageAltNames(kustomizedImage)
	if err != nil {
		return nil, errors.Wrap(err, "build image name alts")
	}

	return images, nil
}
