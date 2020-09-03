package image

import (
	"fmt"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	kustomizeimage "sigs.k8s.io/kustomize/api/types"
)

func ImageInfoFromFile(registry registry.RegistryOptions, nameParts []string) (kustomizeimage.Image, error) {
	// imageNameParts looks like this:
	// ["quay.io", "someorg", "imagename", "imagetag"]
	// or
	// ["quay.io", "someorg", "imagename", "sha256", "<sha>"]
	// we want to discard everything upto "imagename" and replace that with local host and namespace

	image := kustomizeimage.Image{}

	if len(nameParts) < 3 {
		return image, errors.Errorf("not enough parts in image name: %v", nameParts)
	}

	newImageNameParts := []string{registry.Endpoint}
	if registry.Namespace != "" {
		newImageNameParts = append(newImageNameParts, registry.Namespace)
	}
	var originalName, tag, separator string
	if nameParts[len(nameParts)-2] == "sha256" {
		newImageNameParts = append(newImageNameParts, nameParts[len(nameParts)-3])
		originalName = path.Join(nameParts[:len(nameParts)-2]...)
		tag = fmt.Sprintf("sha256:%s", nameParts[len(nameParts)-1])
		separator = "@"
		image.Digest = nameParts[len(nameParts)-1]
	} else {
		newImageNameParts = append(newImageNameParts, nameParts[len(nameParts)-2])
		originalName = path.Join(nameParts[:len(nameParts)-1]...)
		tag = fmt.Sprintf("%s", nameParts[len(nameParts)-1])
		separator = ":"
		image.NewTag = tag
	}

	image.Name = fmt.Sprintf("%s%s%s", originalName, separator, tag)
	image.NewName = path.Join(newImageNameParts...)

	return image, nil
}

// DestRef returns the location to push the image to on the dest registry
func DestRef(registry registry.RegistryOptions, srcImage string) string {
	imageParts := strings.Split(srcImage, "/")
	lastPart := imageParts[len(imageParts)-1]

	if registry.Namespace == "" {
		return fmt.Sprintf("%s/%s", registry.Endpoint, lastPart)
	}
	return fmt.Sprintf("%s/%s/%s", registry.Endpoint, registry.Namespace, lastPart)
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
func destImageName(registry registry.RegistryOptions, srcImage string) string {
	imageParts := strings.Split(srcImage, "/")
	lastPart := imageParts[len(imageParts)-1]
	lastPart = stripImageTag(lastPart)

	if registry.Namespace == "" {
		return fmt.Sprintf("%s/%s", registry.Endpoint, lastPart)
	}
	return fmt.Sprintf("%s/%s/%s", registry.Endpoint, registry.Namespace, lastPart)
}

func buildImageAlts(destRegistry registry.RegistryOptions, image string) ([]kustomizeimage.Image, error) {
	imgParts := strings.Split(image, "/")
	if len(imgParts) == 1 {
		// this means the image is something like "redis", which refers to "docker.io/library/redis"
		imgParts = append([]string{"docker.io", "library"}, imgParts...)
	} else if len(imgParts) == 2 {
		// this means the image is something like "kotsadm/kotsadm-api", which refers to "docker.io/kotsadm/kotsadm-api"
		imgParts = append([]string{"docker.io"}, imgParts...)
	}

	// if the last substring doesn't contain ':', it is untagged and needs 'latest' appended
	// otherwise, split it on '@' and then ':'
	if strings.Contains(imgParts[len(imgParts)-1], ":") {
		imgParts = append(imgParts[:len(imgParts)-1], strings.Split(imgParts[len(imgParts)-1], "@")...)
		imgParts = append(imgParts[:len(imgParts)-1], strings.Split(imgParts[len(imgParts)-1], ":")...)
	} else {
		imgParts = append(imgParts, "latest")
	}

	imageInfo, err := ImageInfoFromFile(destRegistry, imgParts)
	if err != nil {
		return nil, errors.Wrap(err, "get image info")
	}

	newName := destImageName(destRegistry, image)
	imageWithoutTag := stripImageTag(image)

	var newImages []kustomizeimage.Image
	firstImage := kustomizeimage.Image{
		Name:    imageWithoutTag,
		NewName: newName,
		NewTag:  imageInfo.NewTag,
		Digest:  imageInfo.Digest,
	}
	newImages = append(newImages, firstImage)

	return newImages, nil
}
