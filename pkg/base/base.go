package base

import (
	"gopkg.in/yaml.v2"
)

type Base struct {
	Files []BaseFile
}

type BaseFile struct {
	Path    string
	Content []byte
}

type OverlySimpleGVK struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

// ShouldBeIncludedInBase attempts to determine if this is a valid Kubernetes manifest.
// It accomplished this by trying to unmarshal the YAML and looking for a apiVersion and Kind
// It currently cannot return an error
func (f BaseFile) ShouldBeIncludedInBase(excludeKotsKinds bool) (bool, error) {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(f.Content, &o); err != nil {
		return false, nil
	}

	if o.APIVersion == "" || o.Kind == "" {
		return false, nil
	}

	if excludeKotsKinds {
		if o.APIVersion == "kots.io/v1beta1" {
			return false, nil
		}

		if o.APIVersion == "troubleshoot.replicated.com/v1beta1" {
			return false, nil
		}
	}

	return true, nil
}
