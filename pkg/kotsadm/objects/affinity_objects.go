package kotsadm

import (
	corev1 "k8s.io/api/core/v1"
)

func defaultKotsNodeAffinity() *corev1.NodeAffinity {
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
					},
				},
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/arch",
							Operator: corev1.NodeSelectorOpNotIn,
							Values: []string{
								"arm64",
							},
						},
					},
				},
			},
		},
	}
}
