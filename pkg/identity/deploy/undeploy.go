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
	if err := deleteSecret(ctx, clientset, namespace, namePrefix); err != nil {
		return errors.Wrap(err, "failed to delete secret")
	}
	return nil
}

func deleteIngress(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) error {
	err := clientset.ExtensionsV1beta1().Ingresses(namespace).Delete(ctx, prefixName(namePrefix, "dex"), metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete ingress")
}

func deleteService(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) error {
	err := clientset.CoreV1().Services(namespace).Delete(ctx, prefixName(namePrefix, "dex"), metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete service")
}

func deleteDeployment(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) error {
	err := clientset.AppsV1().Deployments(namespace).Delete(ctx, prefixName(namePrefix, "dex"), metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete deployment")
}

func deleteSecret(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) error {
	err := clientset.CoreV1().Secrets(namespace).Delete(ctx, prefixName(namePrefix, "dex"), metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete secret")
}
