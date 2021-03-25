package kotsadm

import (
	"bytes"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getPostgresYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var statefulset bytes.Buffer
	if deployOptions.PostgresPassword == "" {
		deployOptions.PostgresPassword = uuid.New().String()
	}

	size, err := getSize(deployOptions, "postgres", resource.MustParse("1Gi"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get size")
	}

	if err := s.Encode(kotsadmobjects.PostgresStatefulset(deployOptions, size), &statefulset); err != nil {
		return nil, errors.Wrap(err, "failed to marshal postgres statefulset")
	}
	docs["postgres-statefulset.yaml"] = statefulset.Bytes()

	var service bytes.Buffer
	if err := s.Encode(kotsadmobjects.PostgresService(deployOptions.Namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal postgres service")
	}
	docs["postgres-service.yaml"] = service.Bytes()

	return docs, nil
}

func ensurePostgres(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensurePostgresSecret(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres secret")
	}

	size, err := getSize(deployOptions, "postgres", resource.MustParse("1Gi"))
	if err != nil {
		return errors.Wrap(err, "failed to get size")
	}

	if err := ensurePostgresStatefulset(deployOptions, clientset, size); err != nil {
		return errors.Wrap(err, "failed to ensure postgres statefulset")
	}

	if err := ensurePostgresService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres service")
	}

	return nil
}

func ensurePostgresStatefulset(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, size resource.Quantity) error {
	_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Get(context.TODO(), "kotsadm-postgres", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing statefulset")
		}

		_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Create(context.TODO(), kotsadmobjects.PostgresStatefulset(deployOptions, size), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create postgres statefulset")
		}
	}

	return nil
}

func ensurePostgresService(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), "kotsadm-postgres", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), kotsadmobjects.PostgresService(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
	}

	return nil
}

func waitForHealthyPostgres(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	log := logger.NewCLILogger()

	log.ChildActionWithSpinner("Waiting for datastore to be ready")
	defer log.FinishChildSpinner()

	start := time.Now()
	for {
		pods, err := clientset.CoreV1().Pods(deployOptions.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=kotsadm-postgres"})
		if err != nil {
			return errors.Wrap(err, "failed to list pods")
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning {
				return nil
			}
		}

		time.Sleep(time.Second)

		if time.Now().Sub(start) > time.Duration(deployOptions.Timeout) {
			return &types.ErrorTimeout{Message: "timeout waiting for postgres pod"}
		}
	}
}
