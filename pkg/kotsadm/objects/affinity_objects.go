package kotsadm

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func defaultKOTSNodeAffinity() *corev1.NodeAffinity {
	return &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/os",
							Operator: corev1.NodeSelectorOpIn,
							Values: []string{
								"linux",
							},
						},
						{
							Key:      "kubernetes.io/arch",
							Operator: corev1.NodeSelectorOpIn,
							Values: []string{
								"amd64",
							},
						},
					},
				},
			},
		},
	}
}

func DefaultKOTSNodeLabelSelector() (labels.Selector, error) {
	osReq, err := labels.NewRequirement("kubernetes.io/os", selection.In, []string{"linux"})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create os requirement")
	}

	archReq, err := labels.NewRequirement("kubernetes.io/arch", selection.NotIn, []string{"arm64"})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create arch requirement")
	}

	selector := labels.NewSelector()
	selector = selector.Add(*osReq, *archReq)

	return selector, nil
}
