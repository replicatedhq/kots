package midstream

import (
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	corev1 "k8s.io/api/core/v1"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type Midstream struct {
	Kustomization *kustomizetypes.Kustomization
	Base          *base.Base
	DocForPatches []k8sdoc.K8sDoc
	PullSecret    *corev1.Secret
}

func CreateMidstream(b *base.Base, images []kustomizetypes.Image, objects []k8sdoc.K8sDoc, pullSecret *corev1.Secret) (*Midstream, error) {
	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		Bases:                 []string{},
		Resources:             []string{},
		Patches:               []kustomizetypes.Patch{},
		PatchesStrategicMerge: []kustomizetypes.PatchStrategicMerge{},
		Images:                images,
	}

	m := Midstream{
		Kustomization: &kustomization,
		Base:          b,
		DocForPatches: objects,
		PullSecret:    pullSecret,
	}

	return &m, nil
}
