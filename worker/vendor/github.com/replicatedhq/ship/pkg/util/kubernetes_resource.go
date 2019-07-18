package util

import (
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kustomize/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/pkg/gvk"
	"sigs.k8s.io/kustomize/pkg/resid"
	"sigs.k8s.io/kustomize/pkg/resource"
)

func NewKubernetesResource(in []byte) (*resource.Resource, error) {
	resourceFactory := resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl())

	resources, err := resourceFactory.SliceFromBytes(in)
	if err != nil {
		return nil, errors.Wrap(err, "decode resource")
	}
	if len(resources) != 1 {
		return nil, fmt.Errorf("expected 1 resource, got %d", len(resources))
	}
	return resources[0], nil
}

func NewKubernetesResources(in []byte) ([]*resource.Resource, error) {
	resourceFactory := resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl())

	resources, err := resourceFactory.SliceFromBytes(in)
	if err != nil {
		return nil, errors.Wrap(err, "decode resources")
	}

	return resources, nil
}

func ResIDs(in []*resource.Resource) (generated []resid.ResId) {
	for _, thisResource := range in {
		generated = append(generated, thisResource.Id())
	}
	return
}

func ToGroupVersionKind(in gvk.Gvk) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   in.Group,
		Version: in.Version,
		Kind:    in.Kind,
	}
}
