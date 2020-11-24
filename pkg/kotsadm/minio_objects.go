package kotsadm

import (
	"fmt"
	"os"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	"github.com/replicatedhq/kots/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func minioStatefulset(deployOptions types.DeployOptions) *appsv1.StatefulSet {
	size := resource.MustParse("4Gi")

	if deployOptions.LimitRange != nil {
		var allowedMax *resource.Quantity
		var allowedMin *resource.Quantity

		for _, limit := range deployOptions.LimitRange.Spec.Limits {
			if limit.Type == corev1.LimitTypePersistentVolumeClaim {
				max, ok := limit.Max["storage"]
				if ok {
					allowedMax = &max
				}

				min, ok := limit.Min["storage"]
				if ok {
					allowedMin = &min
				}
			}
		}

		newSize := promptForSizeIfNotBetween("minio", &size, allowedMin, allowedMax)
		if newSize == nil {
			os.Exit(-1)
		}

		size = *newSize
	}

	var securityContext corev1.PodSecurityContext
	if !deployOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(1001),
			FSGroup:   util.IntPointer(1001),
		}
	}

	var pullSecrets []corev1.LocalObjectReference
	if s := kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.KotsadmOptions); s != nil {
		pullSecrets = []corev1.LocalObjectReference{
			{
				Name: s.ObjectMeta.Name,
			},
		}
	}

	statefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-minio",
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "kotsadm-minio",
						Labels: types.GetKotsadmLabels(),
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName(corev1.ResourceStorage): size,
							},
						},
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: types.GetKotsadmLabels(map[string]string{
						"app": "kotsadm-minio",
					}),
					Annotations: map[string]string{
						"backup.velero.io/backup-volumes": "kotsadm-minio,minio-config-dir",
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext:  &securityContext,
					ImagePullSecrets: pullSecrets,
					Volumes: []corev1.Volume{
						{
							Name: "kotsadm-minio",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "kotsadm-minio",
								},
							},
						},
						{
							Name: "minio-config-dir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Image:           fmt.Sprintf("%s/minio:%s", kotsadmversion.KotsadmRegistry(deployOptions.KotsadmOptions), kotsadmversion.KotsadmTag(deployOptions.KotsadmOptions)),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "kotsadm-minio",
							Command: []string{
								"/bin/sh",
								"-ce",
								"/usr/bin/docker-entrypoint.sh minio -C /home/minio/.minio/ --quiet server /export",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "service",
									ContainerPort: 9000,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kotsadm-minio",
									MountPath: "/export",
								},
								{
									Name:      "minio-config-dir",
									MountPath: "/home/minio/.minio/",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "MINIO_ACCESS_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-minio",
											},
											Key: "accesskey",
										},
									},
								},
								{
									Name: "MINIO_SECRET_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-minio",
											},
											Key: "secretkey",
										},
									},
								},
								{
									Name:  "MINIO_BROWSER",
									Value: "on",
								},
								{
									Name:  "MINIO_UPDATE",
									Value: "off",
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								FailureThreshold:    3,
								SuccessThreshold:    1,
								PeriodSeconds:       30,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/minio/health/live",
										Port:   intstr.FromString("service"),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								FailureThreshold:    3,
								SuccessThreshold:    1,
								PeriodSeconds:       15,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/minio/health/ready",
										Port:   intstr.FromString("service"),
										Scheme: corev1.URISchemeHTTP,
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
			},
		},
	}

	return statefulset
}

func minioService(namespace string) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm-minio",
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "service",
					Port:       9000,
					TargetPort: intstr.FromInt(9000),
				},
			},
		},
	}

	return service
}
