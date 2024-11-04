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

// removeNodeAPI should be removable when we don't need to support direct upgrade paths from 1.19.6 and before
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

// removeNodeAPIRBAC should be removable when we don't need to support direct upgrade paths from 1.19.6 and before
func removeNodeAPIRBAC(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	isClusterScoped, err := isKotsadmClusterScoped(deployOptions)
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

// removeNodeAPIClusterRBAC should be removable when we don't need to support direct upgrade paths from 1.19.6 and before
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

func getAPIAutoCreateClusterToken(namespace string, cli kubernetes.Interface) (string, error) {
	autoCreateClusterTokenSecretVal, err := getAPIClusterToken(namespace, cli)
	if err != nil {
		return "", errors.Wrap(err, "failed to get autocreate cluster token from secret")
	} else if autoCreateClusterTokenSecretVal != "" {
		return autoCreateClusterTokenSecretVal, nil
	}

	container, err := getKotsadmContainer(namespace, cli)
	if err != nil {
		return "", errors.Wrap(err, "failed to get kotsadm container")
	}

	for _, env := range container.Env {
		if env.Name == "AUTO_CREATE_CLUSTER_TOKEN" {
			return env.Value, nil
		}
	}

	return "", errors.New("failed to find autocreateclustertoken env on api statefulset")
}

func getKotsInstallID(namespace string, cli kubernetes.Interface) (string, error) {
	configMap, err := cli.CoreV1().ConfigMaps(namespace).Get(context.TODO(), types.KotsadmConfigMap, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return "", errors.Wrap(err, "failed to read configmap")
	}
	if err == nil && configMap.Data != nil {
		if installID, ok := configMap.Data["kots-install-id"]; ok {
			return installID, nil
		}
	}

	// configmap does not exist or does not have the installation id, check deployment or statefulset

	container, err := getKotsadmContainer(namespace, cli)
	if err != nil {
		return "", errors.Wrap(err, "failed to get kotsadm container")
	}

	for _, env := range container.Env {
		if env.Name == "KOTS_INSTALL_ID" {
			return env.Value, nil
		}
	}

	// don't fail since there are installs out there that don't have this stored in an env var or the config map because either:
	// - they were installed with an older version of KOTS before this was added
	// - they were affected by a bug that removed the env var on upgrade
	return "", nil
}

func getHTTPProxySettings(namespace string, cli kubernetes.Interface) (httpProxy, httpsProxy, noProxy string, err error) {
	container, err := getKotsadmContainer(namespace, cli)
	if err != nil {
		return "", "", "", errors.Wrap(err, "failed to get kotsadm container")
	}

	for _, env := range container.Env {
		if env.Name == "HTTP_PROXY" {
			httpProxy = env.Value
		}
		if env.Name == "HTTPS_PROXY" {
			httpsProxy = env.Value
		}
		if env.Name == "NO_PROXY" {
			noProxy = env.Value
		}
	}

	return httpProxy, httpsProxy, noProxy, nil
}

func hasStrictSecurityContext(namespace string, cli kubernetes.Interface) (bool, error) {
	podSpec, err := getKotsadmPodSpec(namespace, cli)
	if err != nil {
		return false, errors.Wrap(err, "failed to get kotsadm pod spec")
	}

	if podSpec.SecurityContext == nil {
		return false, nil
	}
	if podSpec.SecurityContext.RunAsNonRoot == nil {
		return false, nil
	}

	return *podSpec.SecurityContext.RunAsNonRoot, nil
}

func getKotsadmPodSpec(namespace string, cli kubernetes.Interface) (*corev1.PodSpec, error) {
	deploy, err := cli.AppsV1().Deployments(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err == nil {
		return &deploy.Spec.Template.Spec, nil
	} else if !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get deployment")
	}

	sts, err := cli.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get statefulset")
	}
	return &sts.Spec.Template.Spec, nil
}

func getKotsadmContainer(namespace string, cli kubernetes.Interface) (*corev1.Container, error) {
	podSpec, err := getKotsadmPodSpec(namespace, cli)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm pod spec")
	}
	for _, c := range podSpec.Containers {
		if c.Name == "kotsadm" {
			return &c, nil
		}
	}
	return nil, errors.New("failed to find kotsadm container")
}
