package resources

import (
	"context"

	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func EnsureKotsadmRole(namespace string, clientset kubernetes.Interface) error {
	role := kotsadmobjects.KotsadmRole(namespace)

	currentRole, err := clientset.RbacV1().Roles(namespace).Get(context.TODO(), "kotsadm-role", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get role")
		}

		_, err := clientset.RbacV1().Roles(namespace).Create(context.TODO(), role, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create role")
		}
		return nil
	}

	currentRole = updateKotsadmRole(currentRole, role)

	// we have now changed the role, so an upgrade is required
	_, err = clientset.RbacV1().Roles(namespace).Update(context.TODO(), currentRole, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update role")
	}

	return nil
}

func updateKotsadmRole(existing, desiredRole *rbacv1.Role) *rbacv1.Role {
	existing.Rules = desiredRole.Rules

	return existing
}

func EnsureKotsadmRoleBinding(roleBindingNamespace string, kotsadmNamespace string, clientset kubernetes.Interface) error {
	roleBinding := kotsadmobjects.KotsadmRoleBinding(roleBindingNamespace, kotsadmNamespace)

	currentRoleBinding, err := clientset.RbacV1().RoleBindings(roleBindingNamespace).Get(context.TODO(), "kotsadm-rolebinding", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get rolebinding")
		}

		_, err := clientset.RbacV1().RoleBindings(roleBindingNamespace).Create(context.TODO(), roleBinding, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create rolebinding")
		}
		return nil
	}

	currentRoleBinding = updateKotsadmRoleBinding(currentRoleBinding, roleBinding)

	// we have now changed the rolebinding, so an upgrade is required
	_, err = clientset.RbacV1().RoleBindings(roleBindingNamespace).Update(context.TODO(), currentRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update rolebinding")
	}

	return nil
}

func updateKotsadmRoleBinding(existing, desiredRoleBinding *rbacv1.RoleBinding) *rbacv1.RoleBinding {
	existing.Subjects = desiredRoleBinding.Subjects

	return existing
}
