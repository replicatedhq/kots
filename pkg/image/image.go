package image

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/v3/pkg/image"
)

func BuildRewriteList(rootDir string, host string, namespace string) ([]image.Image, error) {
	images, err := findImages(rootDir, host, namespace, []string{})
	return images, errors.Wrap(err, "failed to find images")
}

func findImages(srcDir string, host string, namespace string, imageNameParts []string) ([]image.Image, error) {
	files, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list image files")
	}

	images := make([]image.Image, 0)
	for _, file := range files {
		if file.IsDir() {
			moreImages, err := findImages(filepath.Join(srcDir, file.Name()), host, namespace, append(imageNameParts, file.Name()))
			if err != nil {
				return nil, err // no error wrapping because this is a recursive call
			}
			images = append(images, moreImages...)
		} else {
			image, err := imageInfoFromFile(host, namespace, append(imageNameParts, file.Name()))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to create local image name")
			}
			images = append(images, image)
		}
	}

	return images, nil
}

func imageInfoFromFile(registryHost string, namespace string, nameParts []string) (image.Image, error) {
	// imageNameParts looks like this:
	// ["quay.io", "someorg", "imagename", "imagetag"]
	// or
	// ["quay.io", "someorg", "imagename", "sha256", "<sha>"]
	// we want to discard everything upto "imagename" and replace that with local host and namespace

	image := image.Image{}

	if len(nameParts) < 4 {
		return image, fmt.Errorf("not enough parts in image name: %v", nameParts)
	}

	newImageNameParts := []string{registryHost, namespace}
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
