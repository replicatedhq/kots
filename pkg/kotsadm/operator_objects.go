package kotsadm

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func operatorClusterRole() *rbacv1.ClusterRole {
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kotsadm-operator-role",
			Labels: types.GetKotsadmLabels(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     metav1.Verbs{"*"},
			},
		},
	}

	return clusterRole
}

func operatorClusterRoleBinding(serviceAccountNamespace string) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kotsadm-operator-rolebinding",
			Labels: types.GetKotsadmLabels(),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "kotsadm-operator",
				Namespace: serviceAccountNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "kotsadm-operator-role",
		},
	}

	return clusterRoleBinding
}

func operatorRole(namespace string) *rbacv1.Role {
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-operator-role",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
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
			Labels:    types.GetKotsadmLabels(),
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
			Labels:    types.GetKotsadmLabels(),
		},
	}

	return serviceAccount
}

func updateOperatorDeployment(deployment *appsv1.Deployment, deployOptions types.DeployOptions) error {
	var securityContext corev1.PodSecurityContext
	if !deployOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(1001),
		}
	}

	desiredDeployment := operatorDeployment(deployOptions)

	// ensure the non-optional kots labels are present (added in 1.11.0)
	if deployment.ObjectMeta.Labels == nil {
		deployment.ObjectMeta.Labels = map[string]string{}
	}
	deployment.ObjectMeta.Labels[types.KotsadmKey] = types.KotsadmLabelValue
	deployment.ObjectMeta.Labels[types.VeleroKey] = types.VeleroLabelConsoleValue
	if deployment.Spec.Template.ObjectMeta.Labels == nil {
		deployment.Spec.Template.ObjectMeta.Labels = map[string]string{}
	}
	deployment.Spec.Template.ObjectMeta.Labels[types.KotsadmKey] = types.KotsadmLabelValue
	deployment.Spec.Template.ObjectMeta.Labels[types.VeleroKey] = types.VeleroLabelConsoleValue

	// security context (added in 1.11.0)
	deployment.Spec.Template.Spec.SecurityContext = &securityContext
	containerIdx := -1
	for idx, c := range deployment.Spec.Template.Spec.Containers {
		if c.Name == "kotsadm-operator" {
			containerIdx = idx
		}
	}

	if containerIdx == -1 {
		return errors.New("failed to find kotsadm-operator container in deployment")
	}

	// image
	deployment.Spec.Template.Spec.Containers[containerIdx].Image = fmt.Sprintf("%s/kotsadm-operator:%s", kotsadmRegistry(deployOptions.KotsadmOptions), kotsadmTag(deployOptions.KotsadmOptions))

	// copy the env vars from the desired to existing. this could undo a change that the user had.
	// we don't know which env vars we set and which are user edited. this method avoids deleting
	// env vars that the user added, but doesn't handle edited vars
	mergedEnvs := []corev1.EnvVar{}
	for _, env := range desiredDeployment.Spec.Template.Spec.Containers[0].Env {
		mergedEnvs = append(mergedEnvs, env)
	}
	for _, existingEnv := range deployment.Spec.Template.Spec.Containers[containerIdx].Env {
		isUnxpected := true
		for _, env := range desiredDeployment.Spec.Template.Spec.Containers[0].Env {
			if env.Name == existingEnv.Name {
				isUnxpected = false
			}
		}

		if isUnxpected {
			mergedEnvs = append(mergedEnvs, existingEnv)
		}
	}
	deployment.Spec.Template.Spec.Containers[containerIdx].Env = mergedEnvs

	return nil
}

func operatorDeployment(deployOptions types.DeployOptions) *appsv1.Deployment {
	var securityContext corev1.PodSecurityContext
	if !deployOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(1001),
		}
	}

	var pullSecrets []corev1.LocalObjectReference
	if s := kotsadmPullSecret(deployOptions.Namespace, deployOptions.KotsadmOptions); s != nil {
		pullSecrets = []corev1.LocalObjectReference{
			{
				Name: s.ObjectMeta.Name,
			},
		}
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-operator",
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-operator",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: types.GetKotsadmLabels(map[string]string{
						"app": "kotsadm-operator",
					}),
				},
				Spec: corev1.PodSpec{
					SecurityContext:    &securityContext,
					ServiceAccountName: "kotsadm-operator",
					RestartPolicy:      corev1.RestartPolicyAlways,
					ImagePullSecrets:   pullSecrets,
					Containers: []corev1.Container{
						{
							Image:           fmt.Sprintf("%s/kotsadm-operator:%s", kotsadmRegistry(deployOptions.KotsadmOptions), kotsadmTag(deployOptions.KotsadmOptions)),
							ImagePullPolicy: corev1.PullAlways,
							Name:            "kotsadm-operator",
							Env: []corev1.EnvVar{
								{
									Name:  "KOTSADM_API_ENDPOINT",
									Value: fmt.Sprintf("http://kotsadm-api-node.%s.svc.cluster.local:3000", deployOptions.Namespace),
								},
								{
									Name: "KOTSADM_TOKEN",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: types.ClusterTokenSecret,
											},
											Key: types.ClusterTokenSecret,
										},
									},
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
