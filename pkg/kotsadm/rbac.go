package kotsadm

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ensureRBAC(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensureRBACAnalyze(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure analyze rbac")
	}

	if deployOptions.IncludeShip {
		if err := ensureRBACShipInit(deployOptions.Namespace, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure ship init rbac")
		}

		if err := ensureRBACShipWatch(deployOptions.Namespace, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure ship watch rbac")
		}

		if err := ensureRBACShipUpdate(deployOptions.Namespace, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure ship update rbac")
		}

		if err := ensureRBACShipEdit(deployOptions.Namespace, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure ship edit rbac")
		}
	}

	return nil
}

func ensureRBACAnalyze(namespace string, clientset *kubernetes.Clientset) error {
	name := "kotsadm-analyze"

	if err := ensureServiceAccount(namespace, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create analyze service account")
	}

	if err := ensureRole(namespace, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create analyze service account")
	}
	if err := ensureRoleBinding(namespace, name, name, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create analyze service account")
	}

	return nil
}

func ensureRBACShipInit(namespace string, clientset *kubernetes.Clientset) error {
	name := "kotsadm-shipinit"

	if err := ensureServiceAccount(namespace, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create shipinit service account")
	}

	if err := ensureRole(namespace, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create shipinit service account")
	}
	if err := ensureRoleBinding(namespace, name, name, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create shipinit service account")
	}

	return nil
}

func ensureRBACShipWatch(namespace string, clientset *kubernetes.Clientset) error {
	name := "kotsadm-shipwatch"

	if err := ensureServiceAccount(namespace, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create shipwatch service account")
	}
	if err := ensureRole(namespace, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create shipwatch service account")
	}
	if err := ensureRoleBinding(namespace, name, name, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create shipwatch service account")
	}

	return nil
}

func ensureRBACShipUpdate(namespace string, clientset *kubernetes.Clientset) error {
	name := "kotsadm-shipupdate"

	if err := ensureServiceAccount(namespace, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create shipupdate service account")
	}

	if err := ensureRole(namespace, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create shipupdate service account")
	}
	if err := ensureRoleBinding(namespace, name, name, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create shipupdate service account")
	}

	return nil
}

func ensureRBACShipEdit(namespace string, clientset *kubernetes.Clientset) error {
	name := "kotsadm-shipedit"

	if err := ensureServiceAccount(namespace, name, clientset); err != nil {
		return errors.Wrap(err, "failed to crfeate shipedit service account")
	}

	if err := ensureRole(namespace, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create shipedit service account")
	}
	if err := ensureRoleBinding(namespace, name, name, name, clientset); err != nil {
		return errors.Wrap(err, "failed to create shipedit service account")
	}

	return nil
}

func ensureServiceAccount(namespace string, name string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get service account")
		}

		serviceAccount := &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Secrets: []corev1.ObjectReference{
				{
					APIVersion: "v1",
					Kind:       "Secret",
					Name:       name,
					Namespace:  namespace,
				},
			},
		}

		_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(serviceAccount)
		if err != nil {
			return errors.Wrap(err, "failed to create service account")
		}
	}

	return nil
}

func ensureRole(namespace string, name string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().Roles(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get role")
		}

		role := &rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Role",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{
						"namespaces",
						"pods",
						"services",
						"secrets",
					},
					Verbs: metav1.Verbs{"list", "get", "create"},
				},
			},
		}

		_, err := clientset.RbacV1().Roles(namespace).Create(role)
		if err != nil {
			return errors.Wrap(err, "failed to create role")
		}
	}

	return nil
}

func ensureRoleBinding(namespace string, name string, roleName string, serviceAccountName string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().RoleBindings(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get rolebinding")
		}

		roleBinding := &rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Role",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccountName,
					Namespace: namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     roleName,
			},
		}

		_, err := clientset.RbacV1().RoleBindings(namespace).Create(roleBinding)
		if err != nil {
			return errors.Wrap(err, "failed to create rolebinding")
		}
	}

	return nil
}
