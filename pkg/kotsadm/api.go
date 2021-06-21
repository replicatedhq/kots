package kotsadm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
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

	if err := removeNodeAPIRBAC(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api rbac")
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
		return "", errors.Wrap(err, "get autocreate cluter token from secret")
	} else if autoCreateClusterTokenSecretVal != "" {
		return autoCreateClusterTokenSecretVal, nil
	}

	existingStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to read statefulset")
	}

	containerIdx := -1
	for idx, c := range existingStatefulSet.Spec.Template.Spec.Containers {
		if c.Name == "kotsadm" {
			containerIdx = idx
		}
	}

	if containerIdx == -1 {
		return "", errors.New("failed to find kotsadm container in statefulset")
	}

	for _, env := range existingStatefulSet.Spec.Template.Spec.Containers[containerIdx].Env {
		if env.Name == "AUTO_CREATE_CLUSTER_TOKEN" {
			return env.Value, nil
		}
	}

	return "", errors.New("failed to find autocreateclustertoken env on api statefulset")
}
