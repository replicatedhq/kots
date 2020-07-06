package kotsadm

import (
	"bytes"
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getMigrationsYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var pod bytes.Buffer
	if err := s.Encode(migrationsPod(deployOptions), &pod); err != nil {
		return nil, errors.Wrap(err, "failed to marshal migrations pod")
	}
	docs["migrations.yaml"] = pod.Bytes()

	return docs, nil
}

func runSchemaHeroMigrations(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	// we don't deploy the operator because that would require too high of
	// a priv. so we just deploy database migrations here, at deployment time

	// find a ready postgres container
	log := logger.NewLogger()

	log.ChildActionWithSpinner("Waiting for datastore to be ready")
	_, err := waitForHealthyPostgres(deployOptions.Namespace, clientset, deployOptions.Timeout)
	if err != nil {
		return errors.Wrap(err, "failed to find healthy postgres pod")
	}
	log.FinishChildSpinner()

	// Deploy the migration pod with an informer attached to clean it up
	if err := createSchemaHeroPod(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to create schemahero pod")
	}

	return nil
}

func waitForHealthyPostgres(namespace string, clientset *kubernetes.Clientset, timeout time.Duration) (string, error) {
	start := time.Now()

	for {
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=kotsadm-postgres"})
		if err != nil {
			return "", errors.Wrap(err, "failed to list pods")
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning {
				return pod.Name, nil
			}
		}

		time.Sleep(time.Second)

		if time.Now().Sub(start) > time.Duration(timeout) {
			return "", &types.ErrorTimeout{Message: "timeout waiting for postgres pod"}
		}
	}
}

func createSchemaHeroPod(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Pods(deployOptions.Namespace).Create(context.TODO(), migrationsPod(deployOptions), metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create pod")
	}

	return nil
}
