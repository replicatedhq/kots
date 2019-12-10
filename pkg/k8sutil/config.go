package k8sutil

import (
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func GetCurrentRules(kubeconfig, context string, clientset *kubernetes.Clientset) ([]rbacv1.PolicyRule, error) {
	masterURL := "" // TODO: this would be set via CLI in kubectl

	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{ClusterInfo: api.Cluster{Server: masterURL}}).RawConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get raw config")
	}

	if context == "" {
		context = rawConfig.CurrentContext
	}
	c, found := rawConfig.Contexts[context]
	if !found {
		return nil, errors.Errorf("no context: %q", context)
	}

	bindings, err := clientset.RbacV1().RoleBindings(c.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list role bindings")
	}

	roleRefs := make([]rbacv1.RoleRef, 0)
	for _, b := range bindings.Items {
		for _, s := range b.Subjects {
			if s.Name == c.AuthInfo {
				roleRefs = append(roleRefs, b.RoleRef)
			}
		}
	}

	rules := make([]rbacv1.PolicyRule, 0)
	for _, roleRef := range roleRefs {
		role, err := clientset.RbacV1().Roles(c.Namespace).Get(roleRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get role")
		}
		rules = append(rules, role.Rules...)
	}

	return rules, nil
}
