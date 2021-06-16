package downstream

import (
	"github.com/replicatedhq/kots/pkg/midstream"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type Downstream struct {
	Kustomization *kustomizetypes.Kustomization
	Midstream     *midstream.Midstream
}

func CreateDownstream(m *midstream.Midstream) (*Downstream, error) {
	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
	}

	d := Downstream{
		Kustomization: &kustomization,
		Midstream:     m,
	}

	return &d, nil
}
