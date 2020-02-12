package k8sutil

import (
	"github.com/replicatedhq/kots/pkg/util"
	rbacv1 "k8s.io/api/rbac/v1"
)

func UpdateRole(existingRole *rbacv1.Role, desiredRole *rbacv1.Role) {
	newRules := []rbacv1.PolicyRule{}

	// merge roles, with any rules only present in the desired role being added to the existing role
	for _, desiredRule := range desiredRole.Rules {
		foundMatch := false
		for _, existingRule := range existingRole.Rules {
			if util.CompareStringArrays(desiredRule.Resources, existingRule.Resources) &&
				util.CompareStringArrays(desiredRule.APIGroups, existingRule.APIGroups) &&
				util.CompareStringArrays(desiredRule.NonResourceURLs, existingRule.NonResourceURLs) &&
				util.CompareStringArrays(desiredRule.ResourceNames, existingRule.ResourceNames) &&
				util.CompareStringArrays(desiredRule.Verbs, existingRule.Verbs) {
				foundMatch = true
			}
		}
		if !foundMatch {
			newRules = append(newRules, desiredRule)
		}
	}

	existingRole.Rules = append(existingRole.Rules, newRules...)
}

func UpdateClusterRole(existingRole *rbacv1.ClusterRole, desiredRole *rbacv1.ClusterRole) {
	newRules := []rbacv1.PolicyRule{}

	// merge roles, with any rules only present in the desired role being added to the existing role
	for _, desiredRule := range desiredRole.Rules {
		foundMatch := false
		for _, existingRule := range existingRole.Rules {
			if util.CompareStringArrays(desiredRule.Resources, existingRule.Resources) &&
				util.CompareStringArrays(desiredRule.APIGroups, existingRule.APIGroups) &&
				util.CompareStringArrays(desiredRule.NonResourceURLs, existingRule.NonResourceURLs) &&
				util.CompareStringArrays(desiredRule.ResourceNames, existingRule.ResourceNames) &&
				util.CompareStringArrays(desiredRule.Verbs, existingRule.Verbs) {
				foundMatch = true
			}
		}
		if !foundMatch {
			newRules = append(newRules, desiredRule)
		}
	}

	existingRole.Rules = append(existingRole.Rules, newRules...)
}
