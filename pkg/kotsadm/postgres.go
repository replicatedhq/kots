package kotsadm

import (
	"bytes"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getPostgresYAML(namespace string, password string) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var statefulset bytes.Buffer
	if password == "" {
		password = uuid.New().String()
	}
	if err := s.Encode(postgresStatefulset(namespace, password), &statefulset); err != nil {
		return nil, errors.Wrap(err, "failed to marshal postgres statefulset")
	}
	docs["postgres-statefulset.yaml"] = statefulset.Bytes()

	var service bytes.Buffer
	if err := s.Encode(postgresService(namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal postgres service")
	}
	docs["postgres-service.yaml"] = service.Bytes()

	return docs, nil
}

func ensurePostgres(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensurePostgresStatefulset(deployOptions.Namespace, deployOptions.PostgresPassword, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres statefulset")
	}

	if err := ensurePostgresService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres service")
	}

	return nil
}

func ensurePostgresStatefulset(namespace string, password string, clientset *kubernetes.Clientset) error {
	_, err := clientset.AppsV1().StatefulSets(namespace).Get("kotsadm-postgres", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing statefulset")
		}

		_, err := clientset.AppsV1().StatefulSets(namespace).Create(postgresStatefulset(namespace, password))
		if err != nil {
			return errors.Wrap(err, "failed to create postgres statefulset")
		}
	}

	return nil
}

func ensurePostgresService(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(namespace).Get("kotsadm-postgres", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(postgresService(namespace))
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
	}

	return nil
}
