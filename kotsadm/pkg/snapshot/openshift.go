package snapshot

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type k8sGetter interface {
	GetClientSet() (kubernetes.Interface, error)
	IsOpenShift(clientset kubernetes.Interface) bool
	GetRole(namespace, roleName string, clientset kubernetes.Interface) (*rbacv1.Role, error)
	GetRoleBinding(namespace, roleBindingName string, clientset kubernetes.Interface) (*rbacv1.RoleBinding, error)
}

type filterGetter struct {
}

func (g *filterGetter) GetClientSet() (kubernetes.Interface, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	return clientset, nil
}

func (g *filterGetter) IsOpenShift(clientset kubernetes.Interface) bool {
	return k8sutil.IsOpenShift(clientset)
}

func (g *filterGetter) GetRole(namespace, roleName string, clientset kubernetes.Interface) (*rbacv1.Role, error) {
	role, err := clientset.RbacV1().Roles(namespace).Get(context.TODO(), roleName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get role")
	}
	return role, nil
}

func (g *filterGetter) GetRoleBinding(namespace, roleBindingName string, clientset kubernetes.Interface) (*rbacv1.RoleBinding, error) {
	binding, err := clientset.RbacV1().RoleBindings(namespace).Get(context.TODO(), roleBindingName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rolebinding")
	}
	return binding, nil
}

// Some warnings in OpenShift clusters are harmless and should not be shown to the user
func filterWarnings(restore *velerov1.Restore, warnings []types.SnapshotError, k8sGetter k8sGetter) ([]types.SnapshotError, error) {
	clientset, err := k8sGetter.GetClientSet()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	if !k8sGetter.IsOpenShift(clientset) {
		return warnings, nil
	}

	// OpenShift maintains a second copy of all Roles and RoleBindings under a different schema prefix
	// They are a mirror copy of the roles and bindings that applications create.
	// When one is deleted, the other one is deleted automatically.
	//
	// Velero will discover these roles and binding and include them in the snapshot.
	// During the restore, they are the first one that get created and that causes the second copy to be created immediately.
	// Later in the restore process, Velero will fail to restore the duplicates.
	filtered := make([]types.SnapshotError, 0)
	for _, warning := range warnings {
		kind, name := parseOpenShiftWarning(warning)
		switch kind {
		case "role", "roles":
			role, err := k8sGetter.GetRole(warning.Namespace, name, clientset)
			if err != nil {
				if kuberneteserrors.IsNotFound(errors.Cause(err)) {
					continue
				}
				return nil, errors.Wrap(err, "failed to get role")
			}
			if role.CreationTimestamp.After(restore.CreationTimestamp.Time) {
				continue
			}
		case "rolebinding", "rolebindings":
			roleBinding, err := k8sGetter.GetRoleBinding(warning.Namespace, name, clientset)
			if err != nil {
				if kuberneteserrors.IsNotFound(errors.Cause(err)) {
					continue
				}
				return nil, errors.Wrap(err, "failed to get rolebinding")
			}
			if roleBinding.CreationTimestamp.After(restore.CreationTimestamp.Time) {
				continue
			}
		}

		filtered = append(filtered, warning)
	}

	return filtered, nil
}

func parseOpenShiftWarning(warning types.SnapshotError) (string, string) {
	// Only two strings we need to parse here at this time, so it's super dumb code
	// could not restore, rolebindings.rbac.authorization.k8s.io "swimlane-backup-binding" already exists. Warning: the in-cluster version is different than the backed-up version.
	// could not restore, roles.rbac.authorization.k8s.io "swimlane-backup" already exists. Warning: the in-cluster version is different than the backed-up version.

	if !strings.HasPrefix(warning.Message, "could not restore, ") {
		return "", ""
	}

	subStr := strings.TrimPrefix(warning.Message, "could not restore, ")
	parts := strings.Split(subStr, " ")
	if len(parts) < 3 {
		return "", ""
	}

	schemaStr := parts[0]
	objectName := parts[1]

	parts = strings.Split(schemaStr, ".")
	objectKind := parts[0]

	objectName = strings.Trim(objectName, "\"")

	return objectKind, objectName
}
