package generator

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func (g *Generator) shipImage() (string, corev1.PullPolicy) {
	imageName := "replicated/ship:latest"
	imagePullPolicy := corev1.PullIfNotPresent
	for _, image := range g.instance.Spec.Images {
		if image.Image == "replicated/ship" {
			if image.Tag != "" {
				imageName = fmt.Sprintf("replicated/ship:%s", image.Tag)
			}
			if image.ImagePullPolicy != "" {
				imagePullPolicy = corev1.PullPolicy(image.ImagePullPolicy)
			}
		}
	}
	return imageName, imagePullPolicy
}

func (g *Generator) shipToolsImage() (string, corev1.PullPolicy) {
	toolsImageName := "replicated/ship-operator-tools:latest"
	toolsImagePullPolicy := corev1.PullIfNotPresent
	for _, image := range g.instance.Spec.Images {
		if image.Image == "replicated/ship-operator-tools" {
			if image.Tag != "" {
				toolsImageName = fmt.Sprintf("replicated/ship-operator-tools:%s", image.Tag)
			}
			if image.ImagePullPolicy != "" {
				toolsImagePullPolicy = corev1.PullPolicy(image.ImagePullPolicy)
			}
		}
	}

	return toolsImageName, toolsImagePullPolicy
}
