package kotsadm

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/replicatedhq/kots/pkg/util"
)

func minioStatefulset(namespace string) *appsv1.StatefulSet {
	statefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: namespace,
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
						Name: "kotsadm-minio",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("4Gi"),
							},
						},
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "kotsadm-minio",
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: util.IntPointer(1001),
						FSGroup:   util.IntPointer(1001),
					},
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
							Image:           fmt.Sprintf("%s/minio:%s", kotsadmRegistry(), kotsadmTag()),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "kotsadm-minio",
							Command: []string{
								"/bin/sh",
								"-ce",
								"/usr/bin/docker-entrypoint.sh minio -C /home/minio/.minio/ server /export",
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
						},
					},
					InitContainers: []corev1.Container{
						{
							Image:           fmt.Sprintf("%s/minio:%s", kotsadmRegistry(), kotsadmTag()),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "kotsadm-minio",
							Command: []string{
								"/bin/sh",
								"-ce",
								"chown -R minio:minio /export && chown -R minio:minio /home/minio/.minio",
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
