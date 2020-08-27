package kotsadm

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getApiYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var role bytes.Buffer
	if err := s.Encode(apiRole(deployOptions.Namespace), &role); err != nil {
		return nil, errors.Wrap(err, "failed to marshal api role")
	}
	docs["api-role.yaml"] = role.Bytes()

	var roleBinding bytes.Buffer
	if err := s.Encode(apiRoleBinding(deployOptions.Namespace), &roleBinding); err != nil {
		return nil, errors.Wrap(err, "failed to marshal api role binding")
	}
	docs["api-rolebinding.yaml"] = roleBinding.Bytes()

	var serviceAccount bytes.Buffer
	if err := s.Encode(apiServiceAccount(deployOptions.Namespace), &serviceAccount); err != nil {
		return nil, errors.Wrap(err, "failed to marshal api service account")
	}
	docs["api-serviceaccount.yaml"] = serviceAccount.Bytes()

	var service bytes.Buffer
	if err := s.Encode(apiService(deployOptions.Namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal api service")
	}
	docs["api-service.yaml"] = service.Bytes()

	return docs, nil
}

func ensureAPI(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensureApiRBAC(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api rbac")
	}

	if err := ensureApplicationMetadata(*deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure custom branding")
	}

	if err := ensureAPIService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api service")
	}

	return nil
}

func ensureApiRBAC(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	isClusterScoped, err := isKotsadmClusterScoped(deployOptions.ApplicationMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to check if kotsadm api is cluster scoped")
	}

	if isClusterScoped {
		err := ensureApiClusterRBAC(deployOptions, clientset)
		return errors.Wrap(err, "failed to ensure api cluster role")
	}

	if err := ensureApiRole(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api role")
	}

	if err := ensureApiRoleBinding(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api role binding")
	}

	if err := ensureApiServiceAccount(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api service account")
	}

	return nil
}

func ensureApiClusterRBAC(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	err := ensureApiClusterRole(clientset)
	if err != nil {
		return errors.Wrap(err, "failed to ensure api cluster role")
	}

	if err := ensureApiClusterRoleBinding(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api cluster role binding")
	}

	if err := ensureApiServiceAccount(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api service account")
	}

	return nil
}

func ensureApiClusterRole(clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().ClusterRoles().Create(context.TODO(), apiClusterRole(), metav1.CreateOptions{})
	if err == nil || kuberneteserrors.IsAlreadyExists(err) {
		return nil
	}

	return errors.Wrap(err, "failed to create cluster role")
}

func ensureApiClusterRoleBinding(serviceAccountNamespace string, clientset *kubernetes.Clientset) error {
	clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), "kotsadm-api-rolebinding", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		_, err := clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), apiClusterRoleBinding(serviceAccountNamespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create cluster rolebinding")
		}
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to get cluster rolebinding")
	}

	for _, subject := range clusterRoleBinding.Subjects {
		if subject.Namespace == serviceAccountNamespace && subject.Name == "kotsadm-api" && subject.Kind == "ServiceAccount" {
			return nil
		}
	}

	clusterRoleBinding.Subjects = append(clusterRoleBinding.Subjects, rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "kotsadm-api",
		Namespace: serviceAccountNamespace,
	})

	_, err = clientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create cluster rolebinding")
	}

	return nil
}

func ensureApiRole(namespace string, clientset *kubernetes.Clientset) error {
	currentRole, err := clientset.RbacV1().Roles(namespace).Get(context.TODO(), "kotsadm-api-role", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get role")
		}

		_, err := clientset.RbacV1().Roles(namespace).Create(context.TODO(), apiRole(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create role")
		}
		return nil
	}

	// we have now changed the role, so an upgrade is required
	k8sutil.UpdateRole(currentRole, apiRole(namespace))
	_, err = clientset.RbacV1().Roles(namespace).Update(context.TODO(), currentRole, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update role")
	}

	return nil
}

func ensureApiRoleBinding(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().RoleBindings(namespace).Get(context.TODO(), "kotsadm-api-rolebinding", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get rolebinding")
		}

		_, err := clientset.RbacV1().RoleBindings(namespace).Create(context.TODO(), apiRoleBinding(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create rolebinding")
		}
	}

	// We have never changed the role binding, so there is no "upgrade" applied

	return nil
}

func ensureApiServiceAccount(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), "kotsadm-api", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get serviceaccouont")
		}

		_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), apiServiceAccount(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create serviceaccount")
		}
	}

	// We have never changed the service account, so there is no "upgrade" applied

	return nil
}

func ensureAPIDeployment(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	existingDeployment, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get(context.TODO(), "kotsadm-api", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Create(context.TODO(), apiDeployment(deployOptions), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}

		return nil
	}

	_, err = clientset.AppsV1().Deployments(deployOptions.Namespace).Update(context.TODO(), existingDeployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update api deployment")
	}

	return nil
}

func ensureAPIService(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), "kotsadm-api-node", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), apiService(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
	}

	// We have never changed the api service. We renamed it in 1.11.0, but that's a new object creation

	return nil
}

func getAPIAutoCreateClusterToken(namespace string, clientset *kubernetes.Clientset) (string, error) {
	autoCreateClusterTokenSecretVal, err := getAPIClusterToken(namespace, clientset)
	if err != nil {
		return "", errors.Wrap(err, "get autocreate cluter token from secret")
	} else if autoCreateClusterTokenSecretVal != "" {
		return autoCreateClusterTokenSecretVal, nil
	}

	existingDeployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), "kotsadm-api", metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to read deployment")
	}

	containerIdx := -1
	for idx, c := range existingDeployment.Spec.Template.Spec.Containers {
		if c.Name == "kotsadm-api" {
			containerIdx = idx
		}
	}

	if containerIdx == -1 {
		return "", errors.New("failed to find kotsadm-api container in deployment")
	}

	for _, env := range existingDeployment.Spec.Template.Spec.Containers[containerIdx].Env {
		if env.Name == "AUTO_CREATE_CLUSTER_TOKEN" {
			return env.Value, nil
		}
	}

	return "", errors.New("failed to find autocreateclustertoken env on api deployment")
}
