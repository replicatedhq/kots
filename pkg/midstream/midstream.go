package midstream

import (
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/image"
	kustomizetypes "sigs.k8s.io/kustomize/v3/pkg/types"
)

type Midstream struct {
	Kustomization *kustomizetypes.Kustomization
	Base          *base.Base
}

func CreateMidstream(b *base.Base, i []image.Image) (*Midstream, error) {
	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		Bases:  []string{},
		Images: image.ToKustomizationType(i),
	}

	m := Midstream{
		Kustomization: &kustomization,
		Base:          b,
	}

	return &m, nil
}
