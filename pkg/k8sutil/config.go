package k8sutil

import (
	"context"

	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	authorizationv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetCurrentRules(deployOptions kotsadmtypes.DeployOptions, clientset kubernetes.Interface) ([]rbacv1.PolicyRule, error) {
	sar := &authorizationv1.SelfSubjectRulesReview{
		Spec: authorizationv1.SelfSubjectRulesReviewSpec{
			Namespace: deployOptions.Namespace,
		},
	}
	response, err := clientset.AuthorizationV1().SelfSubjectRulesReviews().Create(context.TODO(), sar, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "no SAR in ns %s", deployOptions.Namespace)
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
