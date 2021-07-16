package kotsadm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func removeNodeAPI(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	ns := deployOptions.Namespace

	err := clientset.AppsV1().Deployments(ns).Delete(context.TODO(), "kotsadm-api", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete deployment")
	}

	err = clientset.CoreV1().Services(ns).Delete(context.TODO(), "kotsadm-api-node", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete service")
	}

	if deployOptions.EnsureRBAC {
		if err := removeNodeAPIRBAC(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure api rbac")
		}
	}

	return nil
}

func removeNodeAPIRBAC(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	isClusterScoped, err := isKotsadmClusterScoped(deployOptions.ApplicationMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to check if kotsadm api is cluster scoped")
	}

	if isClusterScoped {
		err := removeNodeAPIClusterRBAC(deployOptions, clientset)
		return errors.Wrap(err, "failed to ensure api cluster role")
	}

	err = clientset.RbacV1().Roles(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-api-role", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete kotsadm-api-role role")
	}

	err = clientset.RbacV1().RoleBindings(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-api-rolebinding", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete rolebinding")
	}

	return nil
}

func removeNodeAPIClusterRBAC(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	err := clientset.CoreV1().ServiceAccounts(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-api", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete api service account")
	}

	err = clientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), "kotsadm-api-rolebinding", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete cluster rolebinding")
	}

	err = clientset.RbacV1().ClusterRoles().Delete(context.TODO(), "kotsadm-api-role", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete cluster role")
	}

	return nil
}

func getAPIAutoCreateClusterToken(namespace string, clientset *kubernetes.Clientset) (string, error) {
	autoCreateClusterTokenSecretVal, err := getAPIClusterToken(namespace, clientset)
	if err != nil {
		return "", errors.Wrap(err, "failed to get autocreate cluster token from secret")
	} else if autoCreateClusterTokenSecretVal != "" {
		return autoCreateClusterTokenSecretVal, nil
	}

	var containers []corev1.Container

	existingDeployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return "", errors.Wrap(err, "failed to read deployment")
	}
	if err == nil {
		containers = existingDeployment.Spec.Template.Spec.Containers
	} else {
		// deployment not found, check if there's a statefulset
		existingStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
		if err != nil {
			return "", errors.Wrap(err, "failed to read statefulset")
		}
		containers = existingStatefulSet.Spec.Template.Spec.Containers
	}

	containerIdx := -1
	for idx, c := range containers {
		if c.Name == "kotsadm" {
			containerIdx = idx
		}
	}

	if containerIdx == -1 {
		return "", errors.New("failed to find kotsadm container in statefulset")
	}

	for _, env := range containers[containerIdx].Env {
		if env.Name == "AUTO_CREATE_CLUSTER_TOKEN" {
			return env.Value, nil
		}
	}

	return "", errors.New("failed to find autocreateclustertoken env on api statefulset")
}
