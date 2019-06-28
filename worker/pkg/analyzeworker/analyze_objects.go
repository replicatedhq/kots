package analyzeworker

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (w *Worker) ensureNamespace(ctx context.Context, namespace *corev1.Namespace) error {
	_, err := w.K8sClient.CoreV1().Namespaces().Get(namespace.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.CoreV1().Namespaces().Create(namespace); err != nil {
			return errors.Wrap(err, "create namespace")
		}
	}

	return nil
}

func (w *Worker) ensureNetworkPolicy(ctx context.Context, networkPolicy *networkv1.NetworkPolicy) error {
	_, err := w.K8sClient.NetworkingV1().NetworkPolicies(networkPolicy.Namespace).Get(networkPolicy.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.NetworkingV1().NetworkPolicies(networkPolicy.Namespace).Create(networkPolicy); err != nil {
			return errors.Wrap(err, "create networkPolicy")
		}
	}

	return nil
}

func (w *Worker) ensureServiceAccount(ctx context.Context, service *corev1.ServiceAccount) error {
	_, err := w.K8sClient.CoreV1().ServiceAccounts(service.Namespace).Get(service.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.CoreV1().ServiceAccounts(service.Namespace).Create(service); err != nil {
			return errors.Wrap(err, "create serviceAccount")
		}
	}

	return nil
}

func (w *Worker) ensureConfigMap(ctx context.Context, configMap *corev1.ConfigMap) error {
	_, err := w.K8sClient.CoreV1().ConfigMaps(configMap.Namespace).Get(configMap.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.CoreV1().ConfigMaps(configMap.Namespace).Create(configMap); err != nil {
			return errors.Wrap(err, "create configMap")
		}
	}

	return nil
}

func (w *Worker) ensurePod(ctx context.Context, pod *corev1.Pod) error {
	_, err := w.K8sClient.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.CoreV1().Pods(pod.Namespace).Create(pod); err != nil {
			return errors.Wrap(err, "create pod")
		}
	}

	return nil
}

func (w *Worker) ensureService(ctx context.Context, service *corev1.Service) error {
	_, err := w.K8sClient.CoreV1().Services(service.Namespace).Get(service.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.CoreV1().Services(service.Namespace).Create(service); err != nil {
			return errors.Wrap(err, "create service")
		}
	}

	return nil
}

func (w *Worker) ensureRole(todo context.Context, role *v1.Role) error {
	_, err := w.K8sClient.CoreV1().Services(role.Namespace).Get(role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.RbacV1().Roles(role.Namespace).Create(role); err != nil {
			return errors.Wrap(err, "create role")
		}
	}

	return nil

}
func (w *Worker) ensureRoleBinding(todo context.Context, roleBinding *v1.RoleBinding) error {
	_, err := w.K8sClient.CoreV1().Services(roleBinding.Namespace).Get(roleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.RbacV1().RoleBindings(roleBinding.Namespace).Create(roleBinding); err != nil {
			return errors.Wrap(err, "create rolebinding")
		}
	}

	return nil
}
