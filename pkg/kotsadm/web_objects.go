package kotsadm

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/replicatedhq/kots/pkg/util"
)

func webConfig(deployOptions DeployOptions) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-web-scripts",
			Namespace: deployOptions.Namespace,
		},
		Data: map[string]string{
			"start-kotsadm-web.sh": fmt.Sprintf(`#!/bin/bash
sed -i 's/###_GRAPHQL_ENDPOINT_###/http:\/\/%s\/graphql/g' /usr/share/nginx/html/index.html
sed -i 's/###_REST_ENDPOINT_###/http:\/\/%s\/api/g' /usr/share/nginx/html/index.html
sed -i 's/###_GITHUB_CLIENT_ID_###/not-supported/g' /usr/share/nginx/html/index.html
sed -i 's/###_SHIPDOWNLOAD_ENDPOINT_###/http:\/\/%s\/api\/v1\/download/g' /usr/share/nginx/html/index.html
sed -i 's/###_SHIPINIT_ENDPOINT_###/http:\/\/%s\/api\/v1\/init\//g' /usr/share/nginx/html/index.html
sed -i 's/###_SHIPUPDATE_ENDPOINT_###/http:\/\/%s\/api\/v1\/update\//g' /usr/share/nginx/html/index.html
sed -i 's/###_SHIPEDIT_ENDPOINT_###/http:\/\/%s\/api\/v1\/edit\//g' /usr/share/nginx/html/index.html
sed -i 's/###_GITHUB_REDIRECT_URI_###/http:\/\/%s\/auth\/github\/callback/g' /usr/share/nginx/html/index.html
sed -i 's/###_GITHUB_INSTALL_URL_###/not-supportetd/g' /usr/share/nginx/html/index.html
sed -i 's/###_INSTALL_ENDPOINT_###/http:\/\/%s\/api\/install/g' /usr/share/nginx/html/index.html

nginx -g "daemon off;"`, deployOptions.Hostname, deployOptions.Hostname,
				deployOptions.Hostname, deployOptions.Hostname, deployOptions.Hostname,
				deployOptions.Hostname, deployOptions.Hostname, deployOptions.Hostname),
		},
	}

	return configMap
}

func webDeployment(deployOptions DeployOptions) *appsv1.Deployment {
	var securityContext corev1.PodSecurityContext
	if !deployOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(101),
			FSGroup:   util.IntPointer(101),
		}
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-web",
			Namespace: deployOptions.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-web",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "kotsadm-web",
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &securityContext,
					Volumes: []corev1.Volume{
						{
							Name: "kotsadm-web-scripts",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									DefaultMode: &executableMode,
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "kotsadm-web-scripts",
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Image:           fmt.Sprintf("%s/kotsadm-web:%s", kotsadmRegistry(), kotsadmTag()),
							ImagePullPolicy: corev1.PullAlways,
							Name:            "kotsadm-web",
							Args: []string{
								"/scripts/start-kotsadm-web.sh",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 3000,
								},
							},
							ReadinessProbe: &corev1.Probe{
								FailureThreshold:    3,
								InitialDelaySeconds: 2,
								PeriodSeconds:       2,
								SuccessThreshold:    1,
								TimeoutSeconds:      1,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(3000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kotsadm-web-scripts",
									MountPath: "/scripts/start-kotsadm-web.sh",
									SubPath:   "start-kotsadm-web.sh",
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

func webService(deployOptions DeployOptions) *corev1.Service {
	port := corev1.ServicePort{
		Name:       "http",
		Port:       3000,
		TargetPort: intstr.FromString("http"),
	}

	serviceType := corev1.ServiceTypeClusterIP
	if deployOptions.ServiceType == "NodePort" {
		serviceType = corev1.ServiceTypeNodePort
		port.NodePort = int32(deployOptions.NodePort)
	} else if deployOptions.ServiceType == "LoadBalancer" {
		serviceType = corev1.ServiceTypeLoadBalancer
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-web",
			Namespace: deployOptions.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm-web",
			},
			Type: serviceType,
			Ports: []corev1.ServicePort{
				port,
			},
		},
	}
	return service
}
