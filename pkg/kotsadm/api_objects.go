package kotsadm

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func apiClusterRole() *rbacv1.ClusterRole {
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kotsadm-api-role",
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

func apiRole(namespace string) *rbacv1.Role {
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-api-role",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
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
				ResourceNames: []string{"kotsadm-encryption", "kotsadm-gitops", auth.KotsadmAuthstringSecretName},
				Verbs:         metav1.Verbs{"get", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     metav1.Verbs{"create"},
			},
		},
	}

	return role
}

func apiClusterRoleBinding(serviceAccountNamespace string) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kotsadm-api-rolebinding",
			Labels: types.GetKotsadmLabels(),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "kotsadm-api",
				Namespace: serviceAccountNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "kotsadm-api-role",
		},
	}

	return clusterRoleBinding
}

func apiRoleBinding(namespace string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-api-rolebinding",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "kotsadm-api",
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "kotsadm-api-role",
		},
	}

	return roleBinding
}

func apiServiceAccount(namespace string) *corev1.ServiceAccount {
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-api",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
	}

	return serviceAccount
}

func updateApiDeployment(deployment *appsv1.Deployment, deployOptions types.DeployOptions) error {
	var securityContext corev1.PodSecurityContext
	if !deployOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(1001),
		}
	}

	desiredDeployment := apiDeployment(deployOptions)

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
		if c.Name == "kotsadm-api" {
			containerIdx = idx
		}
	}

	if containerIdx == -1 {
		return errors.New("failed to find kotsadm-api container in deployment")
	}

	deployment.Spec.Template.Spec.Containers[containerIdx].Image = fmt.Sprintf("%s/kotsadm-api:%s", kotsadmRegistry(deployOptions.KotsadmOptions), kotsadmTag(deployOptions.KotsadmOptions))

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

func apiDeployment(deployOptions types.DeployOptions) *appsv1.Deployment {
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

	env := []corev1.EnvVar{
		{
			Name: "SHARED_PASSWORD_BCRYPT",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-password",
					},
					Key: "passwordBcrypt",
				},
			},
		},
		{
			Name: "AUTO_CREATE_CLUSTER_TOKEN",
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
			Name:  "SHIP_API_ENDPOINT",
			Value: fmt.Sprintf("http://kotsadm.%s.svc.cluster.local:3000", deployOptions.Namespace),
		},
		{
			Name:  "SHIP_API_ADVERTISE_ENDPOINT",
			Value: "http://localhost:8800",
		},
		{
			Name:  "S3_ENDPOINT",
			Value: "http://kotsadm-minio:9000",
		},
		{
			Name:  "S3_BUCKET_NAME",
			Value: "kotsadm",
		},
		{
			Name: "API_ENCRYPTION_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-encryption",
					},
					Key: "encryptionKey",
				},
			},
		},
		{
			Name: "S3_ACCESS_KEY_ID",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-minio",
					},
					Key: "accesskey",
				},
			},
		},
		{
			Name: "S3_SECRET_ACCESS_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-minio",
					},
					Key: "secretkey",
				},
			},
		},
		{
			Name:  "S3_BUCKET_ENDPOINT",
			Value: "true",
		},
		{
			Name: "SESSION_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-session",
					},
					Key: "key",
				},
			},
		},
		{
			Name: "POSTGRES_URI",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-postgres",
					},
					Key: "uri",
				},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}
	env = append(env, getProxyEnv(deployOptions)...)

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-api",
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-api",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: types.GetKotsadmLabels(map[string]string{
						"app": "kotsadm-api",
					}),
				},
				Spec: corev1.PodSpec{
					SecurityContext:    &securityContext,
					ServiceAccountName: "kotsadm-api",
					RestartPolicy:      corev1.RestartPolicyAlways,
					ImagePullSecrets:   pullSecrets,
					Containers: []corev1.Container{
						{
							Image:           fmt.Sprintf("%s/kotsadm-api:%s", kotsadmRegistry(deployOptions.KotsadmOptions), kotsadmTag(deployOptions.KotsadmOptions)),
							ImagePullPolicy: corev1.PullAlways,
							Name:            "kotsadm-api",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 3000,
								},
							},
							ReadinessProbe: &corev1.Probe{
								FailureThreshold:    3,
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(3000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							Env: env,
						},
					},
				},
			},
		},
	}

	return deployment
}

func apiService(namespace string) *corev1.Service {
	port := corev1.ServicePort{
		Name:       "http",
		Port:       3000,
		TargetPort: intstr.FromString("http"),
	}

	serviceType := corev1.ServiceTypeClusterIP

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-api-node",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm-api",
			},
			Type: serviceType,
			Ports: []corev1.ServicePort{
				port,
			},
		},
	}

	return service
}
