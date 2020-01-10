package k8sutil

import (
	"github.com/pkg/errors"
	authorizationv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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

	sar := &authorizationv1.SelfSubjectRulesReview{
		Spec: authorizationv1.SelfSubjectRulesReviewSpec{
			Namespace: c.Namespace,
		},
	}
	response, err := clientset.AuthorizationV1().SelfSubjectRulesReviews().Create(sar)
	if err != nil {
		return nil, errors.Wrapf(err, "no SAR in ns %s", c.Namespace)
	}

	return convertToPolicyRule(response.Status), nil
}

func convertToPolicyRule(status authorizationv1.SubjectRulesReviewStatus) []rbacv1.PolicyRule {
	ret := []rbacv1.PolicyRule{}
	// only include resource rules, not NonResourceRules, as those can't be part of a role
	for _, resource := range status.ResourceRules {
		ret = append(ret, rbacv1.PolicyRule{
			Verbs:         resource.Verbs,
			APIGroups:     resource.APIGroups,
			Resources:     resource.Resources,
			ResourceNames: resource.ResourceNames,
		})
	}
	return ret
}
