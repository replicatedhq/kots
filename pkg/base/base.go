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

func (f BaseFile) ShouldBeIncludedInBase() (bool, error) {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(f.Content, &o); err != nil {
		return false, nil
	}

	if o.APIVersion == "" || o.Kind == "" {
		return false, nil
	}

	return true, nil
}
