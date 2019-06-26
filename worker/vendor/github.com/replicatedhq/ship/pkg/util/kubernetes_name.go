package util

import (
	"fmt"
	"regexp"
)

type MinimalK8sYaml struct {
	Kind     string             `json:"kind" yaml:"kind" hcl:"kind"`
	Metadata MinimalK8sMetadata `json:"metadata" yaml:"metadata" hcl:"metadata"`
}

type MinimalK8sMetadata struct {
	Name      string `json:"name" yaml:"name" hcl:"name"`
	Namespace string `json:"namespace" yaml:"namespace" hcl:"namespace"`
}

func GenerateNameFromMetadata(k8sYaml MinimalK8sYaml, idx int) string {
	fileName := fmt.Sprintf("%s-%d", k8sYaml.Kind, idx)

	if k8sYaml.Metadata.Name != "" {
		fileName = k8sYaml.Kind + "-" + k8sYaml.Metadata.Name
		if k8sYaml.Metadata.Namespace != "" && k8sYaml.Metadata.Namespace != "default" {
			fileName += "-" + k8sYaml.Metadata.Namespace
		}
	}

	fileName = regexp.MustCompile(`[/\\:]`).ReplaceAllString(fileName, "-")

	return fileName
}
