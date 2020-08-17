package kotsadm

import (
	"fmt"
	"time"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func migrationsPod(deployOptions types.DeployOptions) *corev1.Pod {
	name := fmt.Sprintf("kotsadm-migrations-%d", time.Now().Unix())

	var securityContext corev1.PodSecurityContext
	if !deployOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(1001),
			FSGroup:   util.IntPointer(1001),
		}
	}

	var pullSecrets []corev1.LocalObjectReference
	if s := kotsadmPullSecret(deployOptions.Namespace, deployOptions.KotsadmOptions); s != nil {
		pullSecrets = []corev1.LocalObjectReference{
			{
				Name: s.ObjectMeta.Name,
			},
		}
	}

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: defaultKotsNodeAffinity(),
			},
			SecurityContext:  &securityContext,
			RestartPolicy:    corev1.RestartPolicyOnFailure,
			ImagePullSecrets: pullSecrets,
			Volumes: []corev1.Volume{
				{
					Name: "migrations",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							Medium: corev1.StorageMediumMemory,
						},
					},
				},
			},
			InitContainers: []corev1.Container{
				{
					Image:           fmt.Sprintf("%s/kotsadm-migrations:%s", kotsadmRegistry(deployOptions.KotsadmOptions), kotsadmTag(deployOptions.KotsadmOptions)),
					ImagePullPolicy: corev1.PullAlways,
					Name:            "plan",
					Args:            []string{"plan"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "migrations",
							MountPath: "/migrations",
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "SCHEMAHERO_DRIVER",
							Value: "postgres",
						},
						{
							Name:  "SCHEMAHERO_SPEC_FILE",
							Value: "/tables",
						},
						{
							Name:  "SCHEMAHERO_OUT",
							Value: "/migrations/plan.yaml",
						},
						{
							Name: "SCHEMAHERO_URI",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "kotsadm-postgres",
									},
									Key: "uri",
								},
							},
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"cpu":    resource.MustParse("100m"),
							"memory": resource.MustParse("100Mi"),
						},
						Requests: corev1.ResourceList{
							"cpu":    resource.MustParse("50m"),
							"memory": resource.MustParse("50Mi"),
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Image:           fmt.Sprintf("%s/kotsadm-migrations:%s", kotsadmRegistry(deployOptions.KotsadmOptions), kotsadmTag(deployOptions.KotsadmOptions)),
					ImagePullPolicy: corev1.PullAlways,
					Name:            "apply",
					Args:            []string{"apply"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "migrations",
							MountPath: "/migrations",
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "SCHEMAHERO_DRIVER",
							Value: "postgres",
						},
						{
							Name:  "SCHEMAHERO_DDL",
							Value: "/migrations/plan.yaml",
						},
						{
							Name: "SCHEMAHERO_URI",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "kotsadm-postgres",
									},
									Key: "uri",
								},
							},
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"cpu":    resource.MustParse("100m"),
							"memory": resource.MustParse("100Mi"),
						},
						Requests: corev1.ResourceList{
							"cpu":    resource.MustParse("50m"),
							"memory": resource.MustParse("50Mi"),
						},
					},
				},
			},
		},
	}

	return pod
}
