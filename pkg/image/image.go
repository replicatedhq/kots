package image

import (
	"sigs.k8s.io/kustomize/v3/pkg/image"
)

type Image struct {
	Name    string
	NewName string
	NewTag  string
	Digest  string
}

func ToKustomizationType(images []Image) []image.Image {
	result := make([]image.Image, 0)
	for _, i := range images {
		result = append(result, i.ToKustomizationType())
	}
	return result
}

func (i Image) ToKustomizationType() image.Image {
	return image.Image{
		Name:    i.Name,
		NewName: i.NewName,
		NewTag:  i.NewTag,
		Digest:  i.Digest,
	}
}
