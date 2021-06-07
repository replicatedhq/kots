package midstream

import (
	"fmt"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	corev1 "k8s.io/api/core/v1"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type Midstream struct {
	Kustomization  *kustomizetypes.Kustomization
	Base           *base.Base
	DocForPatches  []k8sdoc.K8sDoc
	PullSecret     *corev1.Secret
	IdentitySpec   *kotsv1beta1.Identity
	IdentityConfig *kotsv1beta1.IdentityConfig
}

func CreateMidstream(b *base.Base, images []kustomizetypes.Image, objects []k8sdoc.K8sDoc, pullSecret *corev1.Secret, identitySpec *kotsv1beta1.Identity, identityConfig *kotsv1beta1.IdentityConfig) (*Midstream, error) {
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
		Kustomization:  &kustomization,
		Base:           b,
		DocForPatches:  objects,
		PullSecret:     pullSecret,
		IdentitySpec:   identitySpec,
		IdentityConfig: identityConfig,
	}

	fmt.Println(b.Bases)

	return &m, nil
}
