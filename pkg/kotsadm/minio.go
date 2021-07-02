package kotsadm

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getMinioYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	size, err := getSize(deployOptions, "minio", resource.MustParse("4Gi"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get size")
	}
	minioSts, err := kotsadmobjects.MinioStatefulset(deployOptions, size)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get minio statefulset definition")
	}
	var statefulset bytes.Buffer
	if err := s.Encode(minioSts, &statefulset); err != nil {
		return nil, errors.Wrap(err, "failed to marshal minio statefulset")
	}
	docs["minio-statefulset.yaml"] = statefulset.Bytes()

	var service bytes.Buffer
	if err := s.Encode(kotsadmobjects.MinioService(deployOptions.Namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal minio service")
	}
	docs["minio-service.yaml"] = service.Bytes()

	return docs, nil
}

func ensureMinio(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	size, err := getSize(deployOptions, "minio", resource.MustParse("4Gi"))
	if err != nil {
		return errors.Wrap(err, "failed to get size")
	}

	if err := ensureS3Secret(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio secret")
	}

	if err := ensureMinioStatefulset(deployOptions, clientset, size); err != nil {
		return errors.Wrap(err, "failed to ensure minio statefulset")
	}

	if err := ensureMinioService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio service")
	}

	return nil
}

func ensureMinioStatefulset(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, size resource.Quantity) error {
	desiredMinio, err := kotsadmobjects.MinioStatefulset(deployOptions, size)
	if err != nil {
		return errors.Wrap(err, "failed to get desired minio statefulset definition")
	}

	ctx := context.TODO()
	existingMinio, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Get(ctx, "kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing statefulset")
		}

		_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Create(ctx, desiredMinio, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create minio statefulset")
		}

		return nil
	}

	if len(existingMinio.Spec.Template.Spec.Containers) != 1 || len(desiredMinio.Spec.Template.Spec.Containers) != 1 {
		return errors.New("minio stateful set cannot be upgraded")
	}

	existingMinio.Spec.Template.Spec.Volumes = desiredMinio.Spec.Template.Spec.DeepCopy().Volumes
	existingMinio.Spec.Template.Spec.Containers[0].Image = desiredMinio.Spec.Template.Spec.Containers[0].Image
	existingMinio.Spec.Template.Spec.Containers[0].VolumeMounts = desiredMinio.Spec.Template.Spec.Containers[0].DeepCopy().VolumeMounts

	_, err = clientset.AppsV1().StatefulSets(deployOptions.Namespace).Update(ctx, existingMinio, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update minio statefulset")
	}

	return nil
}

func ensureMinioService(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), "kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), kotsadmobjects.MinioService(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create service")
		}
	}

	return nil
}
