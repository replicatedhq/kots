package kotsadm

import (
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	appsv1 "k8s.io/api/apps/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

func ensureAPI(namespace string, clientset *kubernetes.Clientset) error {
	if err := ensureAPIDeployment(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api deployment")
	}

	if err := ensureAPIService(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api service")
	}

	return nil
}

func ensureAPIDeployment(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.AppsV1().Deployments(namespace).Get("kotsadm-api", metav1.GetOptions{})
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
								Env: []corev1.EnvVar{
									{
										Name:  "AUTO_CREATE_CLUSTER",
										Value: "1",
									},
									{
										Name:  "SHIP_API_ENDPOINT",
										Value: fmt.Sprintf("http://kotsadm-api.%s.svc.cluster.local:3000", namespace),
									},
									{
										Name:  "SHIP_API_ADVERTISE_ENDPOINT",
										Value: fmt.Sprintf("http://kotsadm-api.%s.svc.cluster.local:3000", namespace),
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

		_, err := clientset.AppsV1().Deployments(namespace).Create(deployment)
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}
	}

	return nil
}

func ensureAPIService(namespace string, clientset *kubernetes.Clientset) error {
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
				Name:      "kotsadm-api",
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "kotsadm-api",
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
