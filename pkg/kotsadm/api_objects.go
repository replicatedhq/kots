package kotsadm

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func apiDeployment(namespace string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-api",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-api",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "kotsadm-api",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Image:           "kotsadm/kotsadm-api:alpha",
							ImagePullPolicy: corev1.PullAlways,
							Name:            "kotsadm-api",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 3000,
								},
							},
							ReadinessProbe: &corev1.Probe{
								FailureThreshold:    3,
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(3000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "SHARED_PASSWORD_BCRYPT",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-password",
											},
											Key: "passwordBcrypt",
										},
									},
								},
								{
									Name:  "AUTO_CREATE_CLUSTER",
									Value: "1",
								},
								{
									Name:  "AUTO_CREATE_CLUSTER_NAME",
									Value: "local",
								},
								{
									Name:  "AUTO_CREATE_CLUSTER_TOKEN",
									Value: autoCreateClusterToken,
								},
								{
									Name:  "SHIP_API_ENDPOINT",
									Value: fmt.Sprintf("http://kotsadm-api.%s.svc.cluster.local:3000", namespace),
								},
								{
									Name:  "SHIP_API_ADVERTISE_ENDPOINT",
									Value: "http://localhost:8800",
								},
								{
									Name:  "S3_ENDPOINT",
									Value: "http://kotsadm-minio:9000",
								},
								{
									Name:  "S3_BUCKET_NAME",
									Value: "kotsadm",
								},
								{
									Name: "S3_ACCESS_KEY_ID",
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
									Name: "S3_SECRET_ACCESS_KEY",
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
									Name:  "S3_BUCKET_ENDPOINT",
									Value: "true",
								},
								{
									Name: "SESSION_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-session",
											},
											Key: "key",
										},
									},
								},
								{
									Name: "POSTGRES_URI",
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
						},
					},
				},
			},
		},
	}

	return deployment
}

func apiService(namespace string) *corev1.Service {
	port := corev1.ServicePort{
		Name:       "http",
		Port:       3000,
		TargetPort: intstr.FromString("http"),
	}

	serviceType := corev1.ServiceTypeClusterIP

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-api",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm-api",
			},
			Type: serviceType,
			Ports: []corev1.ServicePort{
				port,
			},
		},
	}

	return service
}
