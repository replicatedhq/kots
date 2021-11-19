package kotsadm

import (
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

func boolPointer(boolValue bool) *bool {
	return &boolValue
}

func securePodContext(user int64, isStrict bool) *corev1.PodSecurityContext {
	var context corev1.PodSecurityContext

	if isStrict {
		context = corev1.PodSecurityContext{
			RunAsNonRoot:       boolPointer(true),
			RunAsUser:          &user,
			RunAsGroup:         &user,
			FSGroup:            &user,
			SupplementalGroups: []int64{user},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		}
	} else {
		context = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(int(user)),
			FSGroup:   util.IntPointer(int(user)),
		}
	}

	return &context
}
func secureContainerContext(isStrict bool) *corev1.SecurityContext {
	var context *corev1.SecurityContext

	if isStrict {
		context = &corev1.SecurityContext{
			Privileged:               boolPointer(false),
			AllowPrivilegeEscalation: boolPointer(false),
			ReadOnlyRootFilesystem:   boolPointer(true),
		}
	}
	return context
}
