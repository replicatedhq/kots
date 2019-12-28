package base

import (
	"crypto/sha256"
	"fmt"

	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
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

type OverlySimpleGVKWithName struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   OverlySimpleMetadata `yaml:"metadata"`
}

type OverlySimpleMetadata struct {
	Name string `yaml:"name"`
}

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
}

func GetGVKWithNameHash(content []byte) []byte {
	o := OverlySimpleGVKWithName{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return nil
	}

	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%s-%s", o.APIVersion, o.Kind, o.Metadata.Name)))
	return h.Sum(nil)
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
