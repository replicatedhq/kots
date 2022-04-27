package deploy

import (
	"context"

	"github.com/pkg/errors"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Undeploy(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) error {
	if err := deleteIngress(ctx, clientset, namespace, namePrefix); err != nil {
		return errors.Wrap(err, "failed to delete ingress")
	}
	if err := deleteService(ctx, clientset, namespace, namePrefix); err != nil {
		return errors.Wrap(err, "failed to delete service")
	}
	if err := deleteDeployment(ctx, clientset, namespace, namePrefix); err != nil {
		return errors.Wrap(err, "failed to delete deployment")
	}
	if err := deleteDexThemeConfigMap(ctx, clientset, namespace, namePrefix); err != nil {
		return errors.Wrap(err, "failed to delete dex theme config map")
	}
	if err := deleteSecret(ctx, clientset, namespace, namePrefix); err != nil {
		return errors.Wrap(err, "failed to delete secret")
	}
	return nil
}

func deleteIngress(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) error {
	err := clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, prefixName(namePrefix, "dex"), metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return nil
	}
	return err
}

func deleteService(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) error {
	err := clientset.CoreV1().Services(namespace).Delete(ctx, prefixName(namePrefix, "dex"), metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return nil
	}
	return err
}

func deleteDeployment(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) error {
	err := clientset.AppsV1().Deployments(namespace).Delete(ctx, prefixName(namePrefix, "dex"), metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return nil
	}
	return err
}

func deleteSecret(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) error {
	err := clientset.CoreV1().Secrets(namespace).Delete(ctx, prefixName(namePrefix, "dex"), metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return nil
	}
	return err
}
