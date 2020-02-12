package kotsadm

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// EnsureAdditionalNamespaces will grant the serviceaccount that kotsadm is running as permissions
// to all namespaces in the additionalNamespaces param. If pruneUnused is true, it will remove
// any namespaces that are not set.
// If running in a non-default namespace, this will convert from a role/rolebinding to a
// clusterole/clusterolebinding when adding additional namespaces, and when pruning back to
// no additional namespaces will convert back to a role/rolebinding.
//
// when we have multiple namespaces, we have a single clusterole with multiple rolebindings
// when we have a single namespace, we have a single role and a rolebinding
func EnsureAdditionalNamespaces(log *logger.Logger, additionalNamespaces []string, kotsNamespace string, pruneUnused bool) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	currentAPIRole, err := clientset.RbacV1().Roles(kotsNamespace).Get("kotsadm-api-role", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get current api role")
	}
	currentKotsadmRole, err := clientset.RbacV1().RoleBindings(kotsNamespace).Get("kotsadm-role", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get current kotsadm role")
	}

	currentAPIClusterRole, err := clientset.RbacV1().ClusterRoles().Get("kotsadm-api-role", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get current api clusterrole")
	}
	currentKotsadmClusterRole, err := clientset.RbacV1().ClusterRoleBindings().Get("kotsadm-role", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get current kotsadm clusterrole")
	}

	desiredNamespaces := []string{
		kotsNamespace,
	}
	desiredNamespaces = append(desiredNamespaces, additionalNamespaces...)

	if len(desiredNamespaces) > 1 {
		for _, desiredNamespace := range desiredNamespaces {
			if err := ensureApiRoleBinding(desiredNamespace, kotsNamespace, clientset); err != nil {
				return errors.Wrap(err, "failed to create kotsadm-api role binding in namespace")
			}
			if err := ensureKotsadmRoleBinding(desiredNamespace, kotsNamespace, clientset); err != nil {
				return errors.Wrap(err, "failed to create kotsadm role binding in namespace")
			}
		}

		if err := ensureKotsadmClusterRole(clientset); err != nil {
			return errors.Wrap(err, "failed to create kotsadm cluster role")
		}

		if err := ensureApiClusterRole(clientset); err != nil {
			return errors.Wrap(err, "failed tocreate kotsadm-api cluster role")
		}

		// delete the role, if it's found
		if currentAPIRole != nil && currentAPIRole.Name != "" {
			if err := clientset.RbacV1().Roles(kotsNamespace).Delete("kotsadm-api-role", &metav1.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "failed to delete kotsadm-api-role")
			}
		}
		if currentKotsadmRole != nil && currentKotsadmRole.Name != "" {
			if err := clientset.RbacV1().Roles(kotsNamespace).Delete("kotsadm-role", &metav1.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "Failed to delete kotsadm-role")
			}
		}
	} else {
		// make this a role, not a cluster role
		if err := ensureKotsadmRole(kotsNamespace, clientset); err != nil {
			return errors.Wrap(err, "failed to create kotsadm role")
		}

		if err := ensureApiRole(kotsNamespace, clientset); err != nil {
			return errors.Wrap(err, "failed to create kotsadm-api role")
		}

		// delete the cluster-roles, if found (is it possible to get here and not have them?)
		if currentAPIClusterRole != nil {
			if err := clientset.RbacV1().ClusterRoles().Delete("kotsadm-api-role", &metav1.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "failed to delete kotsadm-api-role (cluster role)")
			}
		}
		if currentKotsadmClusterRole != nil {
			if err := clientset.RbacV1().ClusterRoles().Delete("kotsadm-role", &metav1.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "failed to delete kotsadm-role (cluster role)")
			}
		}
	}

	if pruneUnused {
		allNamespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to list namespaces to purge from")
		}

		for _, ns := range allNamespaces.Items {
			apiRoleBinding, err := clientset.RbacV1().RoleBindings(ns.Name).Get("kotsadm-api-rolebinding", metav1.GetOptions{})

			if err == nil {
				if len(apiRoleBinding.Subjects) == 1 && apiRoleBinding.Subjects[0].Namespace == kotsNamespace {
					if !util.IsInList(apiRoleBinding.Namespace, desiredNamespaces) {
						if err := clientset.RbacV1().RoleBindings(ns.Name).Delete("kotsadm-api-rolebinding", &metav1.DeleteOptions{}); err != nil {
							return errors.Wrap(err, "failed to delete kotsadm-api-rolebinding rolebinding")
						}
					}
				}
			}

			kotsadmRoleBinding, err := clientset.RbacV1().RoleBindings(ns.Name).Get("kotsadm-rolebinding", metav1.GetOptions{})
			if err == nil {
				if len(kotsadmRoleBinding.Subjects) == 1 && kotsadmRoleBinding.Subjects[0].Namespace == kotsNamespace {
					if !util.IsInList(kotsadmRoleBinding.Namespace, desiredNamespaces) {
						if err := clientset.RbacV1().RoleBindings(ns.Name).Delete("kotsadm-rolebinding", &metav1.DeleteOptions{}); err != nil {
							return errors.Wrap(err, "failed to delete kotsadm-rolebinding rolebinding")
						}
					}
				}
			}
		}
	}

	return nil
}
