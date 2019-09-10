package kotsadm

import (
	"bytes"

	"github.com/pkg/errors"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getOperatorYAML(namespace string) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var role bytes.Buffer
	if err := s.Encode(operatorRole(namespace), &role); err != nil {
		return nil, errors.Wrap(err, "failed to marshal operator role")
	}
	docs["operator-role.yaml"] = role.Bytes()

	var roleBinding bytes.Buffer
	if err := s.Encode(operatorRoleBinding(namespace), &roleBinding); err != nil {
		return nil, errors.Wrap(err, "failed to marshal operator role binding")
	}
	docs["operator-rolebinding.yaml"] = roleBinding.Bytes()

	var serviceAccount bytes.Buffer
	if err := s.Encode(operatorServiceAccount(namespace), &serviceAccount); err != nil {
		return nil, errors.Wrap(err, "failed to marshal operator service account")
	}
	docs["operator-serviceaccount.yaml"] = serviceAccount.Bytes()

	var deployment bytes.Buffer
	if err := s.Encode(operatorDeployment(namespace), &deployment); err != nil {
		return nil, errors.Wrap(err, "failed to marshal operator deployment")
	}
	docs["operator-deployment.yaml"] = deployment.Bytes()

	return docs, nil
}

func ensureOperator(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensureOperatorRBAC(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator rbac")
	}

	if err := ensureOperatorDeployment(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator deployment")
	}

	return nil
}

func ensureOperatorRBAC(namespace string, clientset *kubernetes.Clientset) error {
	if err := ensureOperatorRole(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator role")
	}

	if err := ensureOperatorRoleBinding(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator role binding")
	}

	if err := ensureOperatorServiceAccount(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator service account")
	}

	return nil
}

func ensureOperatorRole(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().Roles(namespace).Get("kotsadm-operator-role", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get role")
		}

		_, err := clientset.RbacV1().Roles(namespace).Create(operatorRole(namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create role")
		}
	}

	return nil
}

func ensureOperatorRoleBinding(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().RoleBindings(namespace).Get("kotsadm-operator-rolebinding", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get rolebinding")
		}

		_, err := clientset.RbacV1().RoleBindings(namespace).Create(operatorRoleBinding(namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create rolebinding")
		}
	}

	return nil
}

func ensureOperatorServiceAccount(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Get("kotsadm-operator", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get serviceaccount")
		}

		_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(operatorServiceAccount(namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create serviceaccount")
		}
	}

	return nil
}

func ensureOperatorDeployment(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get("kotsadm-operator", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		_, err = clientset.AppsV1().Deployments(deployOptions.Namespace).Create(operatorDeployment(deployOptions.Namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}

	}

	return nil
}
