package kotsadm

import corev1 "k8s.io/api/core/v1"

func boolPointer(boolValue bool) *bool {
	return &boolValue
}

func securePodContext(user int64) *corev1.PodSecurityContext {
	context := corev1.PodSecurityContext{
		RunAsNonRoot:       boolPointer(true),
		RunAsUser:          &user,
		RunAsGroup:         &user,
		FSGroup:            &user,
		SupplementalGroups: []int64{user},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	return &context
}
func secureContainerContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               boolPointer(false),
		AllowPrivilegeEscalation: boolPointer(false),
		ReadOnlyRootFilesystem:   boolPointer(true),
	}
}
