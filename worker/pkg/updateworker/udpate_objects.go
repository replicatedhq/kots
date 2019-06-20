package updateworker

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (w *Worker) ensureNamespace(ctx context.Context, namespace *corev1.Namespace) error {
	debug := level.Debug(log.With(w.Logger, "method", "updateworker.Worker.ensureNamespace"))

	debug.Log("event", "get namespace", "namespace", namespace.Name)
	_, err := w.K8sClient.CoreV1().Namespaces().Get(namespace.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		debug.Log("event", "create namespace", "namespace", namespace.Name)
		if _, err := w.K8sClient.CoreV1().Namespaces().Create(namespace); err != nil {
			return errors.Wrap(err, "create namespace")
		}
	}

	return nil
}

func (w *Worker) ensureNetworkPolicy(ctx context.Context, networkPolicy *networkv1.NetworkPolicy) error {
	debug := level.Debug(log.With(w.Logger, "method", "updateworker.Worker.ensureNamespace"))

	debug.Log("event", "get networkPolicy", "networkPolicy", networkPolicy.Name)
	_, err := w.K8sClient.NetworkingV1().NetworkPolicies(networkPolicy.Namespace).Get(networkPolicy.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		debug.Log("event", "create networkPolicy", "networkPolicy", networkPolicy.Name)
		if _, err := w.K8sClient.NetworkingV1().NetworkPolicies(networkPolicy.Namespace).Create(networkPolicy); err != nil {
			return errors.Wrap(err, "create networkPolicy")
		}
	}

	return nil
}

func (w *Worker) ensureSecret(ctx context.Context, secret *corev1.Secret) error {
	debug := level.Debug(log.With(w.Logger, "method", "updateworker.Worker.ensureSecret"))

	debug.Log("event", "get secret", "secret", secret.Name)

	_, err := w.K8sClient.CoreV1().Secrets(secret.Namespace).Get(secret.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		debug.Log("event", "create secret", "secret", secret.Name)
		if _, err := w.K8sClient.CoreV1().Secrets(secret.Namespace).Create(secret); err != nil {
			return errors.Wrap(err, "create secret")
		}
	}

	return nil
}

func (w *Worker) ensureServiceAccount(ctx context.Context, service *corev1.ServiceAccount) error {
	debug := level.Debug(log.With(w.Logger, "method", "updateworker.Worker.ensureSecret"))

	debug.Log("event", "get serviceAccount", "serviceAccount", service.Name)

	_, err := w.K8sClient.CoreV1().ServiceAccounts(service.Namespace).Get(service.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		debug.Log("event", "create serviceAccount", "serviceAccount", service.Name)
		if _, err := w.K8sClient.CoreV1().ServiceAccounts(service.Namespace).Create(service); err != nil {
			return errors.Wrap(err, "create serviceAccount")
		}
	}

	return nil
}

func (w *Worker) ensurePod(ctx context.Context, pod *corev1.Pod) error {
	debug := level.Debug(log.With(w.Logger, "method", "updateworker.Worker.ensurePod"))

	debug.Log("event", "get pod", "pod", pod.Name)
	_, err := w.K8sClient.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		debug.Log("event", "create pod", "pod", pod.Name)
		if _, err := w.K8sClient.CoreV1().Pods(pod.Namespace).Create(pod); err != nil {
			return errors.Wrap(err, "create pod")
		}
	}

	return nil
}

func (w *Worker) ensureService(ctx context.Context, service *corev1.Service) error {
	debug := level.Debug(log.With(w.Logger, "method", "updateworker.Worker.ensureService"))

	debug.Log("event", "get service", "service", service.Name)
	_, err := w.K8sClient.CoreV1().Services(service.Namespace).Get(service.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		debug.Log("event", "create service", "sevice", service.Name)
		if _, err := w.K8sClient.CoreV1().Services(service.Namespace).Create(service); err != nil {
			return errors.Wrap(err, "create service")
		}
	}

	return nil
}
func (w *Worker) ensureRole(todo context.Context, role *v1.Role) error {
	debug := level.Debug(log.With(w.Logger, "method", "updateworker.Worker.ensureRole"))

	debug.Log("event", "get role", "role", role.Name)
	_, err := w.K8sClient.CoreV1().Services(role.Namespace).Get(role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		debug.Log("event", "create role", "role", role.Name)
		if _, err := w.K8sClient.RbacV1().Roles(role.Namespace).Create(role); err != nil {
			return errors.Wrap(err, "create role")
		}
	}

	return nil

}
func (w *Worker) ensureRoleBinding(todo context.Context, roleBinding *v1.RoleBinding) error {
	debug := level.Debug(log.With(w.Logger, "method", "updateworker.Worker.ensureRoleBinding"))

	debug.Log("event", "get rolebinding", "rolebinding", roleBinding.Name)
	_, err := w.K8sClient.CoreV1().Services(roleBinding.Namespace).Get(roleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		debug.Log("event", "create rolebinding", "rolebinding", roleBinding.Name)
		if _, err := w.K8sClient.RbacV1().RoleBindings(roleBinding.Namespace).Create(roleBinding); err != nil {
			return errors.Wrap(err, "create rolebinding")
		}
	}

	return nil
}
