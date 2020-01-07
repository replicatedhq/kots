package client

import (
	"crypto/sha256"
	"fmt"

	"gopkg.in/yaml.v2"
)

type OverlySimpleGVKWithName struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   OverlySimpleMetadata `yaml:"metadata"`
}

type OverlySimpleMetadata struct {
	Name string `yaml:"name"`
}

func GetGVKWithName(content []byte) string {
	o := OverlySimpleGVKWithName{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return ""
	}

	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%s-%s", o.APIVersion, o.Kind, o.Metadata.Name)))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func IsCRD(content []byte) bool {
	o := OverlySimpleGVKWithName{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return false
	}

	return o.APIVersion == "apiextensions.k8s.io/v1beta1" && o.Kind == "CustomResourceDefinition"
}
