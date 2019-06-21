package initworker

import (
	"context"
	"os"
	"os/signal"

	"go.uber.org/zap"
	v1 "k8s.io/api/rbac/v1"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Worker struct {
	Config *config.Config
	Logger *zap.SugaredLogger

	Store     store.Store
	K8sClient kubernetes.Interface
}

func (w *Worker) Run(ctx context.Context) error {
	w.Logger.Infow("starting initworker",
		zap.String("version", version.Version()),
		zap.String("gitSHA", version.GitSHA()),
		zap.Time("buildTime", version.BuildTime()),
	)
	errCh := make(chan error, 2)

	go func() {
		initServer := InitServer{
			Logger: w.Logger,
			Viper:  viper.New(),
			Worker: w,
			Store:  w.Store,
		}

		initServer.Serve(ctx, w.Config.InitServerAddress)
	}()

	go func() {
		err := w.runInformer(context.Background())
		w.Logger.Errorw("intworker informer failed", zap.Error(err))
		errCh <- errors.Wrap(err, "controller ended")
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
		// TODO: possibly cleanup
		return nil
	case err := <-errCh:
		return err
	}
}

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

func (w *Worker) ensureSecret(ctx context.Context, secret *corev1.Secret) error {
	_, err := w.K8sClient.CoreV1().Secrets(secret.Namespace).Get(secret.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.CoreV1().Secrets(secret.Namespace).Create(secret); err != nil {
			return errors.Wrap(err, "create secret")
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
