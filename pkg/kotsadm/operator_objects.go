package kotsadm

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/replicatedhq/kots/pkg/util"
)

func operatorRole(namespace string) *rbacv1.Role {
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-operator-role",
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     metav1.Verbs{"*"},
			},
		},
	}

	return role
}

func operatorRoleBinding(namespace string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-operator-rolebinding",
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "kotsadm-operator",
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "kotsadm-operator-role",
		},
	}

	return roleBinding
}

func operatorServiceAccount(namespace string) *corev1.ServiceAccount {
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-operator",
			Namespace: namespace,
		},
	}

	return serviceAccount
}

func operatorDeployment(namespace, autoCreateClusterToken string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-operator",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-operator",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "kotsadm-operator",
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: util.IntPointer(1001),
					},
					ServiceAccountName: "kotsadm-operator",
					RestartPolicy:      corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Image:           fmt.Sprintf("%s/kotsadm-operator:%s", kotsadmRegistry(), kotsadmTag()),
							ImagePullPolicy: corev1.PullAlways,
							Name:            "kotsadm-operator",
							Env: []corev1.EnvVar{
								{
									Name:  "KOTSADM_API_ENDPOINT",
									Value: fmt.Sprintf("http://kotsadm-api.%s.svc.cluster.local:3000", namespace),
								},
								{
									Name:  "KOTSADM_TOKEN",
									Value: autoCreateClusterToken,
								},
								{
									Name: "KOTSADM_TARGET_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return deployment
}
