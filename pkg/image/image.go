package image

import (
	"fmt"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	kustomizeimage "sigs.k8s.io/kustomize/v3/pkg/image"
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

	newImageNameParts := []string{registry.Endpoint, registry.Namespace}
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

func DestImageName(registry registry.RegistryOptions, srcImage string) string {
	imageParts := strings.Split(srcImage, "/")
	lastPart := imageParts[len(imageParts)-1] // last part

	image := fmt.Sprintf("%s/%s/%s", registry.Endpoint, registry.Namespace, lastPart)

	return image
}

func buildImageAlts(destRegistry registry.RegistryOptions, image string) ([]kustomizeimage.Image, error) {
	imgParts := strings.Split(image, "/")
	if len(imgParts) == 1 {
		imgParts = append([]string{"docker.io", "library"}, imgParts...)
	}
	imageInfo, err := ImageInfoFromFile(destRegistry, imgParts)
	if err != nil {
		return nil, errors.Wrapf(err, "info from %s", image)
	}

	newName := DestImageName(destRegistry, image)

	var newImages []kustomizeimage.Image
	firstImage := kustomizeimage.Image{
		Name:    image,
		NewName: newName,
		NewTag:  imageInfo.NewTag,
		Digest:  imageInfo.Digest,
	}
	newImages = append(newImages, firstImage)

	if strings.HasPrefix(image, "docker.io/library/") {
		image = strings.TrimPrefix(image, "docker.io/library/")
		newImages = append(newImages, kustomizeimage.Image{
			Name:    image,
			NewName: newName,
			NewTag:  imageInfo.NewTag,
			Digest:  imageInfo.Digest,
		})
	}

	if strings.HasSuffix(image, ":latest") {
		image = strings.TrimSuffix(image, ":latest")
		newImages = append(newImages, kustomizeimage.Image{
			Name:    image,
			NewName: newName,
			NewTag:  imageInfo.NewTag,
			Digest:  imageInfo.Digest,
		})
	}

	return newImages, nil
}
