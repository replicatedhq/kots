package k8sutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateRole(t *testing.T) {
	tests := []struct {
		name          string
		existingRole  *rbacv1.Role
		desiredRole   *rbacv1.Role
		expectedRules []rbacv1.PolicyRule
	}{
		{
			name: "apiRuleUpdate", // based on the api rbac policy
			existingRole: &rbacv1.Role{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups:     []string{""},
						Resources:     []string{"configmaps"},
						ResourceNames: []string{"kotsadm-application-metadata", "kotsadm-gitops"},
						Verbs:         metav1.Verbs{"get", "delete", "update"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps"},
						Verbs:     metav1.Verbs{"create"},
					},
					{
						APIGroups:     []string{""},
						Resources:     []string{"secrets"},
						ResourceNames: []string{"kotsadm-encryption", "kotsadm-gitops"},
						Verbs:         metav1.Verbs{"get", "update"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     metav1.Verbs{"create"},
					},
					{
						APIGroups: []string{"blah"},
						Resources: []string{"blah"},
						Verbs:     metav1.Verbs{"blah"}, // this rule is not in the desired set, but should be preserved
					},
				},
			},
			desiredRole: &rbacv1.Role{
				// creation cannot be restricted by name
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups:     []string{""},
						Resources:     []string{"configmaps"},
						ResourceNames: []string{"kotsadm-application-metadata", "kotsadm-gitops"},
						Verbs:         metav1.Verbs{"get", "delete", "update"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps"},
						Verbs:     metav1.Verbs{"create"},
					},
					{
						APIGroups:     []string{""},
						Resources:     []string{"secrets"},
						ResourceNames: []string{"kotsadm-encryption", "kotsadm-gitops", "anotherName"}, // added resource name
						Verbs:         metav1.Verbs{"get", "update"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     metav1.Verbs{"create"},
					},
					{
						APIGroups: []string{"v1"}, // added api group
						Resources: []string{"secrets"},
						Verbs:     metav1.Verbs{"create"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"secrets", "test"}, // added resource type group
						Verbs:     metav1.Verbs{"create"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     metav1.Verbs{"create", "test"}, // added verb
					},
				},
			},
			expectedRules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"configmaps"},
					ResourceNames: []string{"kotsadm-application-metadata", "kotsadm-gitops"},
					Verbs:         metav1.Verbs{"get", "delete", "update"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs:     metav1.Verbs{"create"},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"secrets"},
					ResourceNames: []string{"kotsadm-encryption", "kotsadm-gitops"},
					Verbs:         metav1.Verbs{"get", "update"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     metav1.Verbs{"create"},
				},
				{
					APIGroups: []string{"blah"},
					Resources: []string{"blah"},
					Verbs:     metav1.Verbs{"blah"}, // this rule is not in the desired set, but should be preserved
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"secrets"},
					ResourceNames: []string{"kotsadm-encryption", "kotsadm-gitops", "anotherName"}, // added resource name
					Verbs:         metav1.Verbs{"get", "update"},
				},
				{
					APIGroups: []string{"v1"}, // added api group
					Resources: []string{"secrets"},
					Verbs:     metav1.Verbs{"create"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"secrets", "test"}, // added resource type group
					Verbs:     metav1.Verbs{"create"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     metav1.Verbs{"create", "test"}, // added verb
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			UpdateRole(tt.existingRole, tt.desiredRole)
			req.ElementsMatch(tt.existingRole.Rules, tt.expectedRules)
		})
	}
}
