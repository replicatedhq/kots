package kotsadm

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func titledConfigMap(licenseData []byte) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "titled",
		},
		Data: map[string]string{
			"license.yaml": string(licenseData),
		},
	}

	return configMap
}

func titledService() *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "titled",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "titled",
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "service",
					Port:       3000,
					TargetPort: intstr.FromInt(3000),
				},
			},
		},
	}

	return service
}

func titledDeployment() *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "titled",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "titled",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "titled",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "titled",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "titled",
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Image:           fmt.Sprintf("%s/titled:%s", kotsadmRegistry(), kotsadmTag()),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "titled",
							Ports: []corev1.ContainerPort{
								{
									Name:          "titled",
									ContainerPort: 3000,
								},
							},
							Args: []string{
								"--license-file",
								"/license/license.yaml",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "titled",
									MountPath: "/license/license.yaml",
									SubPath:   "license.yaml",
								},
							},
						},
					},
				},
			},
		},
	}

	return deployment
}
