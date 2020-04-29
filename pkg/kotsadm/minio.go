package kotsadm

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getMinioYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var statefulset bytes.Buffer
	if err := s.Encode(minioStatefulset(deployOptions), &statefulset); err != nil {
		return nil, errors.Wrap(err, "failed to marshal minio statefulset")
	}
	docs["minio-statefulset.yaml"] = statefulset.Bytes()

	var service bytes.Buffer
	if err := s.Encode(minioService(deployOptions.Namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal minio service")
	}
	docs["minio-service.yaml"] = service.Bytes()

	return docs, nil
}

func ensureMinio(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensureS3Secret(deployOptions.Namespace, deployOptions.ObjectStoreOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio secret")
	}

	if err := ensureMinioStatefulset(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio statefulset")
	}

	if err := ensureMinioService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio service")
	}

	return nil
}

func ensureMinioStatefulset(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Get(context.TODO(), "kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing statefulset")
		}

		_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Create(context.TODO(), minioStatefulset(deployOptions), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create minio statefulset")
		}
	}

	return nil
}

func ensureMinioService(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), "kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), minioService(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create service")
		}
	}

	return nil
}
