package client

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

type OverlySimpleGVKWithName struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   OverlySimpleMetadata `yaml:"metadata"`
}

type OverlySimpleMetadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

func GetGVKWithNameAndNs(content []byte, baseNS string) (string, OverlySimpleGVKWithName) {
	o := OverlySimpleGVKWithName{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return "", o
	}

	namespace := baseNS
	if o.Metadata.Namespace != "" {
		namespace = o.Metadata.Namespace
	}

	return fmt.Sprintf("%s-%s-%s-%s", o.APIVersion, o.Kind, o.Metadata.Name, namespace), o
}

func IsCRD(content []byte) bool {
	o := OverlySimpleGVKWithName{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return false
	}

	if o.Kind == "CustomResourceDefinition" {
		return strings.HasPrefix(o.APIVersion, "apiextensions.k8s.io/")
	}

	return false
}

func IsNamespace(content []byte) bool {
	o := OverlySimpleGVKWithName{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return false
	}

	return o.APIVersion == "v1" && o.Kind == "Namespace"
}
