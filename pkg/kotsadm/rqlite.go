package kotsadm

import (
	"bytes"
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getRqliteYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	if deployOptions.RqlitePassword == "" {
		deployOptions.RqlitePassword = uuid.New().String()
	}

	size, err := getSize(deployOptions, "rqlite", resource.MustParse("1Gi"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get size")
	}
	rqliteSts, err := kotsadmobjects.RqliteStatefulset(deployOptions, size)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rqlite statefulset definition")
	}
	var statefulset bytes.Buffer
	if err := s.Encode(rqliteSts, &statefulset); err != nil {
		return nil, errors.Wrap(err, "failed to marshal rqlite statefulset")
	}
	docs["rqlite-statefulset.yaml"] = statefulset.Bytes()

	var service bytes.Buffer
	if err := s.Encode(kotsadmobjects.RqliteService(deployOptions.Namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal rqlite service")
	}
	docs["rqlite-service.yaml"] = service.Bytes()

	var headlessService bytes.Buffer
	if err := s.Encode(kotsadmobjects.RqliteHeadlessService(deployOptions.Namespace), &headlessService); err != nil {
		return nil, errors.Wrap(err, "failed to marshal rqlite headless service")
	}
	docs["rqlite-headless-service.yaml"] = headlessService.Bytes()

	return docs, nil
}

func ensureRqlite(deployOptions types.DeployOptions, clientset kubernetes.Interface) error {
	if err := ensureRqliteSecret(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure rqlite secret")
	}

	size, err := getSize(deployOptions, "rqlite", resource.MustParse("1Gi"))
	if err != nil {
		return errors.Wrap(err, "failed to get size")
	}

	if err := ensureRqliteStatefulset(deployOptions, clientset, size); err != nil {
		return errors.Wrap(err, "failed to ensure rqlite statefulset")
	}

	if err := ensureRqliteService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure rqlite service")
	}

	if err := ensureRqliteHeadlessService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure rqlite headless service")
	}

	return nil
}

func ensureRqliteStatefulset(deployOptions types.DeployOptions, clientset kubernetes.Interface, size resource.Quantity) error {
	desiredRqlite, err := kotsadmobjects.RqliteStatefulset(deployOptions, size)
	if err != nil {
		return errors.Wrap(err, "failed to get desired rqlite statefulset definition")
	}

	ctx := context.TODO()
	existingRqlite, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Get(ctx, "kotsadm-rqlite", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing statefulset")
		}

		_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Create(ctx, desiredRqlite, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create rqlite statefulset")
		}

		return nil
	}

	if len(existingRqlite.Spec.Template.Spec.Containers) != 1 || len(desiredRqlite.Spec.Template.Spec.Containers) != 1 {
		return errors.New("rqlite statefulset cannot be upgraded")
	}

	desiredVolumes := []corev1.Volume{}
	for _, v := range desiredRqlite.Spec.Template.Spec.Volumes {
		desiredVolumes = append(desiredVolumes, *v.DeepCopy())
	}

	desiredVolumeMounts := []corev1.VolumeMount{}
	for _, vm := range desiredRqlite.Spec.Template.Spec.Containers[0].VolumeMounts {
		desiredVolumeMounts = append(desiredVolumeMounts, *vm.DeepCopy())
	}

	existingRqlite.Spec.Template.Spec.Volumes = desiredVolumes
	existingRqlite.Spec.Template.Spec.InitContainers = desiredRqlite.Spec.Template.Spec.InitContainers
	existingRqlite.Spec.Template.Spec.Containers[0].Image = desiredRqlite.Spec.Template.Spec.Containers[0].Image
	existingRqlite.Spec.Template.Spec.Containers[0].VolumeMounts = desiredVolumeMounts
	existingRqlite.Spec.Template.Spec.Containers[0].Env = desiredRqlite.Spec.Template.Spec.Containers[0].Env
	existingRqlite.Spec.Template.Spec.Containers[0].Resources = desiredRqlite.Spec.Template.Spec.Containers[0].Resources

	_, err = clientset.AppsV1().StatefulSets(deployOptions.Namespace).Update(ctx, existingRqlite, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update rqlite statefulset")
	}

	return nil
}

func ensureRqliteService(namespace string, clientset kubernetes.Interface) error {
	_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), "kotsadm-rqlite", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), kotsadmobjects.RqliteService(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
	}

	return nil
}

func ensureRqliteHeadlessService(namespace string, clientset kubernetes.Interface) error {
	_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), "kotsadm-rqlite-headless", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing headless service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), kotsadmobjects.RqliteHeadlessService(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed to create headless service")
		}
	}

	return nil
}
