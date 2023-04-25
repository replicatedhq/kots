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

	// TODO: this is a hack, find a better way to do this.
	// kubernetes does not consider the api version when identifying manifests,
	// and it automatically converts the schema when the api version changes for native k8s objects.
	// kubernetes doesn't/can't handle automatically converting the schema when the api version changes for CRDs though,
	// and vendors would have to implement that themselves using webhook conversion (which is currently pretty complicated).
	// for now, include the api version when identifying CRDs so that they will be re-created to apply the new schema.
	key := fmt.Sprintf("%s-%s-%s", o.Kind, o.Metadata.Name, namespace)
	if isCRD(o) {
		key = fmt.Sprintf("%s-%s", o.APIVersion, key)
	}

	return key, o
}

func IsCRD(content []byte) bool {
	o := OverlySimpleGVKWithName{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return false
	}

	return isCRD(o)
}

func isCRD(o OverlySimpleGVKWithName) bool {
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
