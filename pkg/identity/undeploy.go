package identity

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Undeploy(ctx context.Context, log *logger.Logger, clientset kubernetes.Interface, namespace string) error {
	if err := deleteIngress(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to delete ingress")
	}
	if err := deleteService(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to delete service")
	}
	if err := deleteDeployment(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to delete deployment")
	}
	if err := deleteSecret(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to delete secret")
	}
	if err := deletePostgresSecret(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to delete postgres secret")
	}
	if err := deletePostgresJob(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to delete postgres job")
	}
	return nil
}

func deleteIngress(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	err := clientset.ExtensionsV1beta1().Ingresses(namespace).Delete(ctx, DexIngressName, metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete ingress")
}

func deleteService(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	err := clientset.CoreV1().Services(namespace).Delete(ctx, DexServiceName, metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete service")
}

func deleteDeployment(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	err := clientset.AppsV1().Deployments(namespace).Delete(ctx, DexDeploymentName, metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete deployment")
}

func deleteSecret(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	err := clientset.CoreV1().Secrets(namespace).Delete(ctx, DexSecretName, metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete secret")
}

func deletePostgresSecret(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	err := clientset.CoreV1().Secrets(namespace).Delete(ctx, DexPostgresSecretName, metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete postgres secret")
}

func deletePostgresJob(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	err := clientset.BatchV1().Jobs(namespace).Delete(ctx, DexPostgresJobName, metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete postgres job")
}
