package kotsadm

import (
	"fmt"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ensureOperator(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensureOperatorRBAC(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator rbac")
	}

	if err := ensureOperatorDeployment(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator deployment")
	}

	return nil
}

func ensureOperatorRBAC(namespace string, clientset *kubernetes.Clientset) error {
	if err := ensureOperatorRole(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator role")
	}

	if err := ensureOperatorRoleBinding(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator role binding")
	}

	return nil
}

func ensureOperatorRole(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().Roles(namespace).Get("kotsadm-operator-role", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get role")
		}

		role := &rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
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

		_, err := clientset.RbacV1().Roles(namespace).Create(role)
		if err != nil {
			return errors.Wrap(err, "failed to create role")
		}
	}

	return nil
}

func ensureOperatorRoleBinding(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().RoleBindings(namespace).Get("kotsadm-operator-rolebinding", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get rolebinding")
		}

		roleBinding := &rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Role",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-operator-rolebinding",
				Namespace: namespace,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "default",
					Namespace: namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "kotsadm-operator-role",
			},
		}

		_, err := clientset.RbacV1().RoleBindings(namespace).Create(roleBinding)
		if err != nil {
			return errors.Wrap(err, "failed to create rolebinding")
		}
	}

	return nil
}

func ensureOperatorDeployment(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get("kotsadm-operator", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		deployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-operator",
				Namespace: deployOptions.Namespace,
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
						RestartPolicy: corev1.RestartPolicyAlways,
						Containers: []corev1.Container{
							{
								Image:           "kotsadm/kotsadm-operator:alpha",
								ImagePullPolicy: corev1.PullAlways,
								Name:            "kotsadm-operator",
								Env: []corev1.EnvVar{
									{
										Name:  "KOTSADM_API_ENDPOINT",
										Value: fmt.Sprintf("http://kotsadm-api.%s.svc.cluster.local:3000", deployOptions.Namespace),
									},
									{
										Name:  "KOTSADM_TOKEN",
										Value: autoCreateClusterToken,
									},
								},
							},
						},
					},
				},
			},
		}

		_, err = clientset.AppsV1().Deployments(deployOptions.Namespace).Create(deployment)
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}

	}

	return nil
}
