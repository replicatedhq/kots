package kotsadm

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getOperatorYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var role bytes.Buffer
	if err := s.Encode(operatorRole(deployOptions.Namespace), &role); err != nil {
		return nil, errors.Wrap(err, "failed to marshal operator role")
	}
	docs["operator-role.yaml"] = role.Bytes()

	var roleBinding bytes.Buffer
	if err := s.Encode(operatorRoleBinding(deployOptions.Namespace), &roleBinding); err != nil {
		return nil, errors.Wrap(err, "failed to marshal operator role binding")
	}
	docs["operator-rolebinding.yaml"] = roleBinding.Bytes()

	var serviceAccount bytes.Buffer
	if err := s.Encode(operatorServiceAccount(deployOptions.Namespace), &serviceAccount); err != nil {
		return nil, errors.Wrap(err, "failed to marshal operator service account")
	}
	docs["operator-serviceaccount.yaml"] = serviceAccount.Bytes()

	var deployment bytes.Buffer
	if err := s.Encode(operatorDeployment(deployOptions), &deployment); err != nil {
		return nil, errors.Wrap(err, "failed to marshal operator deployment")
	}
	docs["operator-deployment.yaml"] = deployment.Bytes()

	return docs, nil
}

func ensureOperator(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensureOperatorRBAC(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator rbac")
	}

	if err := ensureOperatorDeployment(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator deployment")
	}

	return nil
}

func ensureOperatorClusterRBAC(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	err := ensureOperatorClusterRole(clientset)
	if err != nil {
		return errors.Wrap(err, "failed to ensure operator cluster role")
	}

	if err := ensureOperatorClusterRoleBinding(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator cluster role binding")
	}

	if err := ensureOperatorServiceAccount(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator service account")
	}

	return nil
}

func ensureOperatorRBAC(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	shouldBeClusterScoped, err := isOperatorClusterScoped(deployOptions.ApplicationMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to check if operator should be cluster scoped")
	}

	// if this is cluster scoped, it's easy... create everything as a cluster role and cluster role binding
	// with pretty open permissions
	if shouldBeClusterScoped {
		return ensureOperatorClusterRBAC(deployOptions, clientset)
	}

	// we want to ensure that the principle of least privilege is applied.
	// so we will create our role and rolebinding
	// and then create a role and role binding PER namespace that the application
	// wants...  everthing will be linked to the same service account

	err = ensureOperatorRole(deployOptions.Namespace, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to ensure operator role")
	}

	if err := ensureOperatorRoleBinding(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator role binding")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(deployOptions.ApplicationMetadata, nil, nil)
	if err != nil {
		return errors.Wrap(err, "failed to decode application metadata")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Application" {
		return errors.New("application metadata contained unepxected gvk")
	}

	application := obj.(*kotsv1beta1.Application)
	for _, additionalNamespace := range application.Spec.AdditionalNamespaces {
		if err = ensureOperatorRole(additionalNamespace, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure operator additional namespace role")
		}

		if err = ensureOperatorRoleBinding(additionalNamespace, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure operator additional namespace role binding")
		}
	}

	if err := ensureOperatorServiceAccount(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator service account")
	}

	return nil
}

func ensureOperatorClusterRole(clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().ClusterRoles().Create(context.TODO(), operatorClusterRole(), metav1.CreateOptions{})
	if err == nil || kuberneteserrors.IsAlreadyExists(err) {
		return nil
	}

	return errors.Wrap(err, "failed to create cluster role")
}

func ensureOperatorRole(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().Roles(namespace).Create(context.TODO(), operatorRole(namespace), metav1.CreateOptions{})
	if err == nil || kuberneteserrors.IsAlreadyExists(err) {
		return nil
	}

	return errors.Wrap(err, "failed to create role")
}

func ensureOperatorClusterRoleBinding(serviceAccountNamespace string, clientset *kubernetes.Clientset) error {
	clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), "kotsadm-operator-rolebinding", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		_, err := clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), operatorClusterRoleBinding(serviceAccountNamespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create cluster rolebinding")
		}
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to get cluster rolebinding")
	}

	for _, subject := range clusterRoleBinding.Subjects {
		if subject.Namespace == serviceAccountNamespace && subject.Name == "kotsadm-operator" && subject.Kind == "ServiceAccount" {
			return nil
		}
	}

	clusterRoleBinding.Subjects = append(clusterRoleBinding.Subjects, rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "kotsadm-operator",
		Namespace: serviceAccountNamespace,
	})

	_, err = clientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create cluster rolebinding")
	}

	return nil
}

func ensureOperatorRoleBinding(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().RoleBindings(namespace).Create(context.TODO(), operatorRoleBinding(namespace), metav1.CreateOptions{})
	if err != nil && !kuberneteserrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create rolebinding")
	}
	return nil
}

func ensureOperatorServiceAccount(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), "kotsadm-operator", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get serviceaccount")
		}

		_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), operatorServiceAccount(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create serviceaccount")
		}
	}

	return nil
}

func ensureOperatorDeployment(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	existingDeployment, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get(context.TODO(), "kotsadm-operator", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		_, err = clientset.AppsV1().Deployments(deployOptions.Namespace).Create(context.TODO(), operatorDeployment(deployOptions), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}

		return nil
	}

	if err = updateOperatorDeployment(existingDeployment, deployOptions); err != nil {
		return errors.Wrap(err, "failed to merge deployment")
	}

	_, err = clientset.AppsV1().Deployments(deployOptions.Namespace).Update(context.TODO(), existingDeployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update operator deployment")
	}

	return nil
}

func isOperatorClusterScoped(applicationMetadata []byte) (bool, error) {
	if applicationMetadata == nil {
		return true, nil
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(applicationMetadata, nil, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to decode application metadata")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Application" {
		return false, errors.New("application metadata contained unepxected gvk")
	}

	application := obj.(*kotsv1beta1.Application)

	// An application can request cluster scope privileges quite simply
	if !application.Spec.RequireMinimalRBACPrivileges {
		return true, nil
	}

	for _, additionalNamespace := range application.Spec.AdditionalNamespaces {
		if additionalNamespace == "*" {
			return true, nil
		}
	}

	return false, nil
}
