package kotsadm

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func runSchemaHeroMigrations(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	// we don't deploy the operator because that would require too high of
	// a priv. so we just deploy database migrations here, at deployment time

	// find a ready postgres container
	log := logger.NewLogger()
	log.ChildActionWithSpinner("Waiting for datastore to be ready")
	_, err := waitForHealthyPostgres(deployOptions.Namespace, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to find health postgres pod")
	}
	log.FinishChildSpinner()

	// Deploy the migration pod with an informer attached to clean it up
	if err := createSchemaHeroPod(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to create schemahero pod")
	}

	return nil
}

func waitForHealthyPostgres(namespace string, clientset *kubernetes.Clientset) (string, error) {
	start := time.Now()

	for {
		pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: "app=kotsadm-postgres"})
		if err != nil {
			return "", errors.Wrap(err, "failed to list pods")
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning {
				return pod.Name, nil
			}
		}

		time.Sleep(time.Second)

		if time.Now().Sub(start) > time.Duration(time.Minute) {
			return "", errors.New("timeout waiting for postgres pod")
		}
	}
}

func createSchemaHeroPod(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	name := fmt.Sprintf("kotsadm-migrations-%d", time.Now().Unix())

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: deployOptions.Namespace,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyOnFailure,
			Containers: []corev1.Container{
				{
					Image:           "kotsadm/kotsadm-migrations:alpha",
					ImagePullPolicy: corev1.PullAlways,
					Name:            name,
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
							Name:  "SCHEMAHERO_URI",
							Value: fmt.Sprintf("postgresql://kotsadm:%s@kotsadm-postgres/kotsadm?connect_timeout=10&sslmode=disable", postgresPassword),
						},
					},
				},
			},
		},
	}

	_, err := clientset.CoreV1().Pods(deployOptions.Namespace).Create(pod)
	if err != nil {
		return errors.Wrap(err, "failed to create pod")
	}

	return nil
}
