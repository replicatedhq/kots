package k8sutil

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func FindKotsadm(clientset *kubernetes.Clientset, namespace string) (string, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=kotsadm"})
	if err != nil {
		return "", errors.Wrap(err, "failed to list pods")
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return pod.Name, nil
		}
	}

	return "", errors.New("unable to find kotsadm pod")
}

func IsKotsadmClusterScoped(ctx context.Context, clientset kubernetes.Interface) bool {
	_, err := clientset.RbacV1().ClusterRoles().Get(ctx, "kotsadm-role", metav1.GetOptions{})
	if err != nil {
		return false
	}
	return true
}

func DeleteKotsadm(clientset *kubernetes.Clientset, namespace string, isKurl bool) error {
	selectorLabels := map[string]string{
		types.KotsadmKey: types.KotsadmLabelValue,
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	}

	// services (does not have a DeleteCollection method)
	services, err := clientset.CoreV1().Services(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list services")
	}
	for _, service := range services.Items {
		err := clientset.CoreV1().Services(namespace).Delete(context.TODO(), service.ObjectMeta.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to delete service %s in namespace %s", service.ObjectMeta.Name, service.ObjectMeta.Namespace)
		}
	}
	if err := waitForDeleteServices(clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete services")
	}

	// deployments
	err = clientset.AppsV1().Deployments(namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm deployments")
	}
	if err := waitForDeleteDeployments(clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete deployments")
	}

	// statefulsets
	err = clientset.AppsV1().StatefulSets(namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm statefulsets")
	}
	if err := waitForDeleteStatefulSets(clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete statefulsets")
	}

	// pods
	err = clientset.CoreV1().Pods(namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm pods")
	}
	if err := waitForDeletePods(clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete pods")
	}

	// PVCs
	err = clientset.CoreV1().PersistentVolumeClaims(namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm PVCs")
	}
	if err := waitForDeletePVCs(clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete PVCs")
	}

	// secrets
	err = clientset.CoreV1().Secrets(namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm secrets")
	}
	if err := waitForDeleteSecrets(clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete secrets")
	}

	// configmaps
	err = clientset.CoreV1().ConfigMaps(namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm configmaps")
	}
	if err := waitForDeleteConfigmaps(clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete configmaps")
	}

	// cluster role bindings
	err = clientset.RbacV1().ClusterRoleBindings().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm clusterrolebindings")
	}
	if err := waitForDeleteClusterRoleBindings(clientset, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete clusterrolebindings")
	}

	// role bindings
	err = clientset.RbacV1().RoleBindings(namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm rolebindings")
	}
	if err := waitForDeleteRoleBindings(clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete rolebindings")
	}

	// cluster roles
	err = clientset.RbacV1().ClusterRoles().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm clusterroles")
	}
	if err := waitForDeleteClusterRoles(clientset, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete clusterroles")
	}

	// roles
	err = clientset.RbacV1().Roles(namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm roles")
	}
	if err := waitForDeleteRoles(clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete roles")
	}

	// service accounts
	err = clientset.CoreV1().ServiceAccounts(namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm serviceaccounts")
	}
	if err := waitForDeleteServiceAccounts(clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete serviceaccounts")
	}

	if !isKurl {
		return nil
	}

	// kURL registry
	registryNS := "kurl"
	err = clientset.AppsV1().Deployments(registryNS).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete registry deployments")
	}
	if err := waitForDeleteDeployments(clientset, registryNS, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete registry deployments")
	}
	err = clientset.CoreV1().Secrets(registryNS).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm secrets")
	}
	if err := waitForDeleteSecrets(clientset, registryNS, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete secrets")
	}

	return nil
}

func waitForDeleteServices(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		services, err := clientset.CoreV1().Services(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list services")
		}
		if len(services.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteDeployments(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list deployments")
		}
		if len(deployments.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteStatefulSets(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list statefulsets")
		}
		if len(statefulsets.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeletePods(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list pods")
		}
		if len(pods.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeletePVCs(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list pvcs")
		}
		if len(pvcs.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteSecrets(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list secrets")
		}
		if len(secrets.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteConfigmaps(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list configmaps")
		}
		if len(configmaps.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteClusterRoleBindings(clientset *kubernetes.Clientset, listOptions metav1.ListOptions) error {
	for {
		crbs, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list clusterrolebindings")
		}
		if len(crbs.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteRoleBindings(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		rbs, err := clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list rolebindings")
		}
		if len(rbs.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteClusterRoles(clientset *kubernetes.Clientset, listOptions metav1.ListOptions) error {
	for {
		crs, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list clusterroles")
		}
		if len(crs.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteRoles(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		roles, err := clientset.RbacV1().Roles(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list roles")
		}
		if len(roles.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteServiceAccounts(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		serviceAccounts, err := clientset.CoreV1().ServiceAccounts(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list serviceaccounts")
		}
		if len(serviceAccounts.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}
