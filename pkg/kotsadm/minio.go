package kotsadm

import (
	"bytes"
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

	var hostpathVolume bytes.Buffer
	if deployOptions.HostNetwork {
		if err := s.Encode(minioHostpathVolume(), &hostpathVolume); err != nil {
			return nil, errors.Wrap(err, "failed to marshal minio hostPath persistent volume")
		}
		docs["minio-pv.yaml"] = hostpathVolume.Bytes()
	}

	var service bytes.Buffer
	if err := s.Encode(minioService(deployOptions.Namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal minio service")
	}
	docs["minio-service.yaml"] = service.Bytes()

	return docs, nil
}

func ensureMinio(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensureS3Secret(deployOptions.Namespace, clientset); err != nil {
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
	_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Get("kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing statefulset")
		}

		_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Create(minioStatefulset(deployOptions))
		if err != nil {
			return errors.Wrap(err, "failed to create minio statefulset")
		}

		if deployOptions.HostNetwork {
			_, err := clientset.CoreV1().PersistentVolumes().Create(minioHostpathVolume())
			if err != nil {
				return errors.Wrap(err, "failed to create minio hostpath persistentvolume")
			}
		}
	}

	return nil
}

func ensureMinioService(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(namespace).Get("kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(minioService(namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create service")
		}
	}

	return nil
}
