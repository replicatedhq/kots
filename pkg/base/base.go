package base

import (
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes/scheme"
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

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

// ShouldBeIncludedInBaseKustomization attempts to determine if this is a valid Kubernetes manifest.
// It accomplished this by trying to unmarshal the YAML and looking for a apiVersion and Kind
func (f BaseFile) ShouldBeIncludedInBaseKustomization(excludeKotsKinds bool) bool {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(f.Content, &o); err != nil {
		return false
	}

	if o.APIVersion == "" || o.Kind == "" {
		return false
	}

	if excludeKotsKinds {
		if o.APIVersion == "kots.io/v1beta1" {
			return false
		}

		if o.APIVersion == "troubleshoot.replicated.com/v1beta1" {
			return false
		}

		// In addition to kotskinds, we exclude the application crd for now
		if o.APIVersion == "app.k8s.io/v1beta1" {
			return false
		}
	}

	return true
}

// ShouldBeIncludedInBaseFilesystem attempts to determine if this is a valid Kubernetes manifest.
// It accomplished this by trying to unmarshal the YAML and looking for a apiVersion and Kind
func (f BaseFile) ShouldBeIncludedInBaseFilesystem(excludeKotsKinds bool) bool {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(f.Content, &o); err != nil {
		return false
	}

	if o.APIVersion == "" || o.Kind == "" {
		return false
	}

	if excludeKotsKinds {
		if o.APIVersion == "kots.io/v1beta1" {
			return false
		}

		if o.APIVersion == "troubleshoot.replicated.com/v1beta1" {
			return false
		}

		if o.APIVersion == "app.k8s.io/v1beta1" {
			return false
		}
	}

	return true
}
