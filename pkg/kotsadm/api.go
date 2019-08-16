package kotsadm

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

var timeoutWaitingForAPI = time.Duration(time.Minute * 2)

func waitForAPI(deployOptions *DeployOptions, clientset *kubernetes.Clientset) error {
	start := time.Now()

	for {
		pods, err := clientset.CoreV1().Pods(deployOptions.Namespace).List(metav1.ListOptions{LabelSelector: "app=kotsadm-api"})
		if err != nil {
			return errors.Wrap(err, "failed to list pods")
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning {
				if pod.Status.ContainerStatuses[0].Ready == true {
					return nil
				}
			}
		}

		time.Sleep(time.Second)

		if time.Now().Sub(start) > timeoutWaitingForAPI {
			return errors.New("timeout waiting for api pod")
		}
	}
}

func ensureAPI(deployOptions *DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensureAPIDeployment(*deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api deployment")
	}

	if err := ensureAPIService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api service")
	}

	return nil
}

func ensureAPIDeployment(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get("kotsadm-api", metav1.GetOptions{})
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
				Namespace: deployOptions.Namespace,
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
										Value: fmt.Sprintf("http://kotsadm-api.%s.svc.cluster.local:3000", deployOptions.Namespace),
									},
									{
										Name:  "SHIP_API_ADVERTISE_ENDPOINT",
										Value: fmt.Sprintf("http://kotsadm-api.%s.svc.cluster.local:3000", deployOptions.Namespace),
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
										Name:  "S3_ACCESS_KEY_ID",
										Value: minioAccessKey,
									},
									{
										Name:  "S3_SECRET_ACCESS_KEY",
										Value: minioSecret,
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

		_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Create(deployment)
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}
	}

	return nil
}

func ensureAPIService(namespace string, clientset *kubernetes.Clientset) error {
	port := corev1.ServicePort{
		Name:       "http",
		Port:       3000,
		TargetPort: intstr.FromString("http"),
	}

	serviceType := corev1.ServiceTypeClusterIP

	_, err := clientset.CoreV1().Services(namespace).Get("kotsadm-api", metav1.GetOptions{})
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
				Type: serviceType,
				Ports: []corev1.ServicePort{
					port,
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
