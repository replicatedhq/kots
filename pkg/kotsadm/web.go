package kotsadm

import (
	"github.com/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

var (
	executableMode = int32(484)
)

func ensureWeb(namespace string, clientset *kubernetes.Clientset) error {
	if err := ensureWebConfig(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure web configmap")
	}

	if err := ensureWebDeployment(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure web deployment")
	}

	if err := ensureWebService(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure web service")
	}

	return nil
}

func ensureWebConfig(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ConfigMaps(namespace).Get("kotsadm-web-scripts", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing config map")
		}

		configMap := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-web-scripts",
				Namespace: namespace,
			},
			Data: map[string]string{
				"start-kotsadm-web.sh": `#!/bin/bash
sed -i 's/###_GRAPHQL_ENDPOINT_###/https:\/\/{{repl ConfigOption "hostname"}}\/graphql/g' /usr/share/nginx/html/index.html
sed -i 's/###_REST_ENDPOINT_###/https:\/\/{{repl ConfigOption "hostname"}}\/api/g' /usr/share/nginx/html/index.html
sed -i 's/###_GITHUB_CLIENT_ID_###/{{repl ConfigOption "github-clientid"}}/g' /usr/share/nginx/html/index.html
sed -i 's/###_SHIPDOWNLOAD_ENDPOINT_###/https:\/\/{{repl ConfigOption "hostname"}}\/api\/v1\/download/g' /usr/share/nginx/html/index.html
sed -i 's/###_SHIPINIT_ENDPOINT_###/https:\/\/{{repl ConfigOption "hostname"}}\/api\/v1\/init\//g' /usr/share/nginx/html/index.html
sed -i 's/###_SHIPUPDATE_ENDPOINT_###/https:\/\/{{repl ConfigOption "hostname"}}\/api\/v1\/update\//g' /usr/share/nginx/html/index.html
sed -i 's/###_SHIPEDIT_ENDPOINT_###/https:\/\/{{repl ConfigOption "hostname"}}\/api\/v1\/edit\//g' /usr/share/nginx/html/index.html
sed -i 's/###_GITHUB_REDIRECT_URI_###/https:\/\/{{repl ConfigOption "hostname"}}\/auth\/github\/callback/g' /usr/share/nginx/html/index.html
sed -i 's/###_GITHUB_INSTALL_URL_###/{{repl ConfigOption "github-installurl" | replace "/" "\\/"}}/g' /usr/share/nginx/html/index.html
sed -i 's/###_INSTALL_ENDPOINT_###/https:\/\/{{repl ConfigOption "hostname"}}\/api\/install/g' /usr/share/nginx/html/index.html

nginx -g "daemon off;"`,
			},
		}

		_, err := clientset.CoreV1().ConfigMaps(namespace).Create(configMap)
		if err != nil {
			return errors.Wrap(err, "failed to create configmap")
		}
	}

	return nil
}

func ensureWebDeployment(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.AppsV1().Deployments(namespace).Get("kotsadm-web", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		deployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-web",
				Namespace: namespace,
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
								Image:           "kotsadm/kotsadm-web:alpha",
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

		_, err := clientset.AppsV1().Deployments(namespace).Create(deployment)
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}
	}

	return nil
}

func ensureWebService(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(namespace).Get("kotsadm-web", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		service := &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-web",
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "kotsadm-web",
				},
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Port:       3000,
						TargetPort: intstr.FromString("http"),
					},
				},
			},
		}

		_, err := clientset.CoreV1().Services(namespace).Create(service)
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
	}

	return nil
}
