package kotsadm

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func restoreJob(backupName string, namespace string, isOpenShift bool, kotsadmOptions types.KotsadmOptions) *batchv1.Job {
	var securityContext corev1.PodSecurityContext
	if !isOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(999),
			FSGroup:   util.IntPointer(999),
		}
	}

	var pullSecrets []corev1.LocalObjectReference
	if s := kotsadmPullSecret(namespace, kotsadmOptions); s != nil {
		pullSecrets = []corev1.LocalObjectReference{
			{
				Name: s.ObjectMeta.Name,
			},
		}
	}

	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-restore",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: types.GetKotsadmLabels(map[string]string{
						"app": "kotsadm-restore",
					}),
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: defaultKotsNodeAffinity(),
					},
					SecurityContext:    &securityContext,
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: "kotsadm",
					ImagePullSecrets:   pullSecrets,
					Containers: []corev1.Container{
						{
							Image:           fmt.Sprintf("%s/kotsadm:%s", kotsadmRegistry(kotsadmOptions), kotsadmTag(kotsadmOptions)),
							ImagePullPolicy: corev1.PullAlways,
							Name:            "kotsadm-restore",
							Command: []string{
								"/kotsadm",
								"restore",
								backupName,
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return job
}
