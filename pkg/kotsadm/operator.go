package kotsadm

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

type RoleScope string

const (
	None      = ""
	Cluster   = "cluster"
	Namespace = "namespace"
)

func getOperatorYAML(namespace, autoCreateClusterToken string) (map[string][]byte, error) {
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
	if err := s.Encode(operatorDeployment(namespace, autoCreateClusterToken), &deployment); err != nil {
		return nil, errors.Wrap(err, "failed to marshal operator deployment")
	}
	docs["operator-deployment.yaml"] = deployment.Bytes()

	return docs, nil
}

func ensureOperator(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	// TODO: log this error on debug level
	rules, _ := k8sutil.GetCurrentRules(deployOptions.Kubeconfig, deployOptions.Context, clientset)

	if err := ensureOperatorRBAC(deployOptions.Namespace, clientset, rules); err != nil {
		return errors.Wrap(err, "failed to ensure operator rbac")
	}

	if err := ensureOperatorDeployment(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator deployment")
	}

	return nil
}

func ensureOperatorRBAC(namespace string, clientset *kubernetes.Clientset, rules []rbacv1.PolicyRule) error {
	scope, err := ensureOperatorRole(namespace, clientset, rules)
	if err != nil {
		return errors.Wrap(err, "failed to ensure operator role")
	}

	if err := ensureOperatorRoleBinding(scope, namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator role binding")
	}

	if err := ensureOperatorServiceAccount(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator service account")
	}

	return nil
}

func ensureOperatorRole(namespace string, clientset *kubernetes.Clientset, rules []rbacv1.PolicyRule) (RoleScope, error) {
	// we'd like to create a cluster scope role, but will settle for namespace scope...

	_, err := clientset.RbacV1().ClusterRoles().Create(operatorClusterRole(namespace))
	if err == nil || kuberneteserrors.IsAlreadyExists(err) {
		return Cluster, nil
	}
	if !kuberneteserrors.IsForbidden(err) {
		return None, errors.Wrap(err, "failed to create cluster role")
	}

	role := operatorRole(namespace)

	_, err = clientset.RbacV1().Roles(namespace).Create(role)
	if err == nil || kuberneteserrors.IsAlreadyExists(err) {
		return Namespace, nil
	}

	// if forbidden, try a more restrictive version
	if !kuberneteserrors.IsForbidden(err) {
		return None, errors.Wrap(err, "failed to create role")
	}

	role.Rules = rules
	_, err = clientset.RbacV1().Roles(namespace).Create(role)
	if err != nil {
		return None, errors.Wrap(err, "failed to create restricted role")
	}

	return Namespace, nil
}

func ensureOperatorRoleBinding(scope RoleScope, namespace string, clientset *kubernetes.Clientset) error {
	if scope == Cluster {
		clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get("kotsadm-operator-rolebinding", metav1.GetOptions{})
		if kuberneteserrors.IsNotFound(err) {
			_, err := clientset.RbacV1().ClusterRoleBindings().Create(operatorClusterRoleBinding(namespace))
			if err != nil {
				return errors.Wrap(err, "failed to create cluster rolebinding")
			}

			return nil
		}

		clusterRoleBinding.Subjects = append(clusterRoleBinding.Subjects, rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      "kotsadm-operator",
			Namespace: namespace,
		})

		_, err = clientset.RbacV1().ClusterRoleBindings().Update(clusterRoleBinding)
		if err != nil {
			return errors.Wrap(err, "failed to create cluster rolebinding")
		}

		return nil
	}

	if scope == Namespace {
		_, err := clientset.RbacV1().RoleBindings(namespace).Create(operatorRoleBinding(namespace))
		if err != nil && !kuberneteserrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "failed to create rolebinding")
		}
		return nil
	}

	return errors.Errorf("failed to create rolebinding for scope %q", scope)
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

		_, err = clientset.AppsV1().Deployments(deployOptions.Namespace).Create(operatorDeployment(deployOptions.Namespace, deployOptions.AutoCreateClusterToken))
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}

	}

	return nil
}
