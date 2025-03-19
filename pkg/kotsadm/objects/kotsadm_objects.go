package kotsadm

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func KotsadmClusterRole() *rbacv1.ClusterRole {
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kotsadm-role",
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

func KotsadmRole(namespace string) *rbacv1.Role {
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-role",
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

func KotsadmClusterRoleBinding(serviceAccountNamespace string) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kotsadm-rolebinding",
			Labels: types.GetKotsadmLabels(),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "kotsadm",
				Namespace: serviceAccountNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "kotsadm-role",
		},
	}

	return clusterRoleBinding
}

func KotsadmRoleBinding(roleBindingNamespace string, kotsadmNamespace string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-rolebinding",
			Namespace: roleBindingNamespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "kotsadm",
				Namespace: kotsadmNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "kotsadm-role",
		},
	}

	return roleBinding
}

func KotsadmServiceAccount(namespace string) *corev1.ServiceAccount {
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
	}

	return serviceAccount
}

func UpdateKotsadmDeployment(existingDeployment *appsv1.Deployment, desiredDeployment *appsv1.Deployment) error {
	containerIdx := -1
	for idx, c := range existingDeployment.Spec.Template.Spec.Containers {
		if c.Name == "kotsadm" {
			containerIdx = idx
		}
	}

	if containerIdx == -1 {
		return errors.New("failed to find kotsadm container in deployment")
	}

	desiredVolumes := []corev1.Volume{}
	for _, v := range desiredDeployment.Spec.Template.Spec.Volumes {
		desiredVolumes = append(desiredVolumes, *v.DeepCopy())
	}

	desiredVolumeMounts := []corev1.VolumeMount{}
	for _, vm := range desiredDeployment.Spec.Template.Spec.Containers[0].VolumeMounts {
		desiredVolumeMounts = append(desiredVolumeMounts, *vm.DeepCopy())
	}

	existingDeployment.Spec.Template.Spec.Volumes = desiredVolumes
	existingDeployment.Spec.Template.Spec.InitContainers = desiredDeployment.Spec.Template.Spec.InitContainers
	existingDeployment.Spec.Template.Spec.Containers[containerIdx].Image = desiredDeployment.Spec.Template.Spec.Containers[0].Image
	existingDeployment.Spec.Template.Spec.Containers[containerIdx].VolumeMounts = desiredVolumeMounts
	existingDeployment.Spec.Template.Spec.Containers[containerIdx].Env = desiredDeployment.Spec.Template.Spec.Containers[0].Env

	updateKotsadmDeploymentScriptsPath(existingDeployment)

	return nil
}

func updateKotsadmDeploymentScriptsPath(existing *appsv1.Deployment) {
	if existing.Spec.Template.Annotations != nil {
		existing.Spec.Template.Annotations["pre.hook.backup.velero.io/command"] = `["/scripts/backup.sh"]`
	}

	for i, c := range existing.Spec.Template.Spec.Containers {
		for j, env := range c.Env {
			if env.Name == "POSTGRES_SCHEMA_DIR" {
				existing.Spec.Template.Spec.Containers[i].Env[j].Value = "/scripts/postgres/tables"
			}
		}
	}

	for i, c := range existing.Spec.Template.Spec.InitContainers {
		if c.Name == "restore-db" {
			existing.Spec.Template.Spec.InitContainers[i].Command = []string{
				"/scripts/restore-db.sh",
			}
		} else if c.Name == "restore-s3" {
			existing.Spec.Template.Spec.InitContainers[i].Command = []string{
				"/scripts/restore-s3.sh",
			}
		}
	}
}

func KotsadmDeployment(deployOptions types.DeployOptions) (*appsv1.Deployment, error) {
	securityContext := k8sutil.SecurePodContext(1001, 1001, deployOptions.StrictSecurityContext)
	if deployOptions.IsOpenShift {
		psc, err := k8sutil.GetOpenShiftPodSecurityContext(deployOptions.Namespace, deployOptions.StrictSecurityContext)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get openshift pod security context")
		}
		securityContext = psc
	}

	var pullSecrets []corev1.LocalObjectReference
	if s := kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.RegistryConfig); s != nil {
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
			Name: "RQLITE_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-rqlite",
					},
					Key: "password",
				},
			},
		},
		{
			Name: "RQLITE_URI",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-rqlite",
					},
					Key: "uri",
				},
			},
		},
		{
			Name: "POSTGRES_URI", // this is still needed for the migration
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-postgres",
					},
					Key:      "uri",
					Optional: pointer.Bool(true),
				},
			},
		},
		{
			Name:  "POSTGRES_SCHEMA_DIR", // this is needed for the migration
			Value: "/scripts/postgres/tables",
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "POD_OWNER_KIND",
			Value: "deployment",
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
			Name:  "API_ENDPOINT",
			Value: fmt.Sprintf("http://kotsadm.%s.svc.cluster.local:3000", deployOptions.Namespace),
		},
		{
			Name:  "API_ADVERTISE_ENDPOINT",
			Value: "http://localhost:8800",
		},
		{
			Name:  "S3_ENDPOINT",
			Value: fmt.Sprintf("http://kotsadm-minio.%s.svc.cluster.local:9000", deployOptions.Namespace),
		},
		{
			Name:  "S3_BUCKET_NAME",
			Value: "kotsadm",
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
	}

	env = append(env, GetProxyEnv(deployOptions)...)

	if d := os.Getenv("DISABLE_KOTSADM_OUTBOUND_CONNECTIONS"); d != "" { // used for e2e testing
		env = append(env, corev1.EnvVar{
			Name:  "DISABLE_OUTBOUND_CONNECTIONS",
			Value: d,
		})
	} else if deployOptions.RegistryConfig.OverrideRegistry != "" || deployOptions.Airgap {
		env = append(env, corev1.EnvVar{
			Name:  "DISABLE_OUTBOUND_CONNECTIONS",
			Value: "true",
		})
	}

	if deployOptions.InstallID != "" {
		env = append(env, corev1.EnvVar{
			Name:  "KOTS_INSTALL_ID",
			Value: deployOptions.InstallID,
		})
	}
	if deployOptions.SimultaneousUploads > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "AIRGAP_UPLOAD_PARALLELISM",
			Value: fmt.Sprintf("%d", deployOptions.SimultaneousUploads),
		})
	}

	if deployOptions.PrivateCAsConfigmap != "" {
		env = append(env, corev1.EnvVar{
			Name:  "SSL_CERT_DIR",
			Value: "/certs",
		})
		env = append(env, corev1.EnvVar{
			Name:  "SSL_CERT_CONFIGMAP",
			Value: deployOptions.PrivateCAsConfigmap,
		})
	}

	podAnnotations := map[string]string{
		"backup.velero.io/backup-volumes":   "backup",
		"pre.hook.backup.velero.io/command": `["/scripts/backup.sh"]`,
		"pre.hook.backup.velero.io/timeout": "10m",
	}
	for k, v := range deployOptions.AdditionalAnnotations {
		podAnnotations[k] = v
	}
	podLabels := map[string]string{
		"app": "kotsadm",
	}
	for k, v := range deployOptions.AdditionalLabels {
		podLabels[k] = v
	}

	volumes := []corev1.Volume{
		{
			Name: "migrations",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
		{
			Name: "backup",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	if deployOptions.PrivateCAsConfigmap != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "kotsadm-private-cas",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: deployOptions.PrivateCAsConfigmap,
					},
				},
			},
		})
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "backup",
			MountPath: "/backup",
		},
		{
			Name:      "tmp",
			MountPath: "/tmp",
		},
	}

	if deployOptions.PrivateCAsConfigmap != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "kotsadm-private-cas",
			MountPath: "/certs",
		})
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "kotsadm",
			Namespace:   deployOptions.Namespace,
			Annotations: deployOptions.AdditionalAnnotations,
			Labels:      types.GetKotsadmLabels(deployOptions.AdditionalLabels),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      types.GetKotsadmLabels(podLabels),
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: defaultKOTSNodeAffinity(),
					},
					Tolerations:        deployOptions.Tolerations,
					SecurityContext:    securityContext,
					Volumes:            volumes,
					ServiceAccountName: "kotsadm",
					RestartPolicy:      corev1.RestartPolicyAlways,
					ImagePullSecrets:   pullSecrets,
					InitContainers: []corev1.Container{
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm-migrations"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "schemahero-plan",
							Args:            []string{"plan"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "migrations",
									MountPath: "/migrations",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "SCHEMAHERO_DRIVER",
									Value: "rqlite",
								},
								{
									Name:  "SCHEMAHERO_SPEC_FILE",
									Value: "/tables",
								},
								{
									Name:  "SCHEMAHERO_OUT",
									Value: "/migrations/plan.yaml",
								},
								{
									Name: "SCHEMAHERO_URI",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-rqlite",
											},
											Key: "uri",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("50m"),
									"memory": resource.MustParse("50Mi"),
								},
							},
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
						},
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm-migrations"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "schemahero-apply",
							Args:            []string{"apply"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "migrations",
									MountPath: "/migrations",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "SCHEMAHERO_DRIVER",
									Value: "rqlite",
								},
								{
									Name:  "SCHEMAHERO_DDL",
									Value: "/migrations/plan.yaml",
								},
								{
									Name: "SCHEMAHERO_URI",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-rqlite",
											},
											Key: "uri",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("50m"),
									"memory": resource.MustParse("50Mi"),
								},
							},
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
						},
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "restore-db",
							Command: []string{
								"/scripts/restore-db.sh",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "backup",
									MountPath: "/backup",
								},
								{
									Name:      "tmp",
									MountPath: "/tmp",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "RQLITE_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-rqlite",
											},
											Key: "password",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("1"),
									"memory": resource.MustParse("2Gi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
							},
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
						},
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "restore-s3",
							Command: []string{
								"/scripts/restore-s3.sh",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "backup",
									MountPath: "/backup",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "S3_ENDPOINT",
									Value: fmt.Sprintf("http://kotsadm-minio.%s.svc.cluster.local:9000", deployOptions.Namespace),
								},
								{
									Name:  "S3_BUCKET_NAME",
									Value: "kotsadm",
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
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("1"),
									"memory": resource.MustParse("2Gi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
							},
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
						},
					},
					Containers: []corev1.Container{
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "kotsadm",
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
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(3000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							VolumeMounts: volumeMounts,
							Env:          env,
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("1"),
									"memory": resource.MustParse("2Gi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
							},
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
						},
					},
				},
			},
		},
	}

	return deployment, nil
}

func UpdateKotsadmStatefulSet(existingStatefulset *appsv1.StatefulSet, desiredStatefulSet *appsv1.StatefulSet) error {
	containerIdx := -1
	for idx, c := range existingStatefulset.Spec.Template.Spec.Containers {
		if c.Name == "kotsadm" {
			containerIdx = idx
		}
	}

	if containerIdx == -1 {
		return errors.New("failed to find kotsadm container in statefulset")
	}

	desiredVolumes := []corev1.Volume{}
	for _, v := range desiredStatefulSet.Spec.Template.Spec.Volumes {
		desiredVolumes = append(desiredVolumes, *v.DeepCopy())
	}

	desiredVolumeMounts := []corev1.VolumeMount{}
	for _, vm := range desiredStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts {
		desiredVolumeMounts = append(desiredVolumeMounts, *vm.DeepCopy())
	}

	existingStatefulset.Spec.Template.Spec.Volumes = desiredVolumes
	existingStatefulset.Spec.Template.Spec.InitContainers = desiredStatefulSet.Spec.Template.Spec.InitContainers
	existingStatefulset.Spec.Template.Spec.Containers[containerIdx].Image = desiredStatefulSet.Spec.Template.Spec.Containers[0].Image
	existingStatefulset.Spec.Template.Spec.Containers[containerIdx].VolumeMounts = desiredVolumeMounts
	existingStatefulset.Spec.Template.Spec.Containers[containerIdx].Env = desiredStatefulSet.Spec.Template.Spec.Containers[0].Env

	updateKotsadmStatefulSetScriptsPath(existingStatefulset)

	return nil
}

func updateKotsadmStatefulSetScriptsPath(existing *appsv1.StatefulSet) {
	if existing.Spec.Template.Annotations != nil {
		existing.Spec.Template.Annotations["pre.hook.backup.velero.io/command"] = `["/scripts/backup.sh"]`
	}

	for i, c := range existing.Spec.Template.Spec.InitContainers {
		if c.Name == "restore-data" {
			existing.Spec.Template.Spec.InitContainers[i].Command = []string{
				"/scripts/restore.sh",
			}
		} else if c.Name == "migrate-s3" {
			existing.Spec.Template.Spec.InitContainers[i].Command = []string{
				"/scripts/migrate-s3.sh",
			}
		}
	}
}

// TODO add configmap for additional CAs
func KotsadmStatefulSet(deployOptions types.DeployOptions, size resource.Quantity) (*appsv1.StatefulSet, error) {
	securityContext := k8sutil.SecurePodContext(1001, 1001, deployOptions.StrictSecurityContext)
	if deployOptions.IsOpenShift {
		// we have to specify a pod security context here because if we don't, here's what will happen:
		// the kotsadm service account is associated with a role/clusterrole that has wildcard privileges,
		// which gives the kotsadm pod/container the permission to run as any user id in openshift.
		// now, since the kotsadm docker image defines user "kotsadm" with uid "1001",
		// openshift will run the container with that user and won't automatically assign a uid and fsgroup.
		// so, if we don't assign an fsgroup, and neither will openshift, the kotsadm pod/container won't have write permissions to the volume mount
		// for the main pvc ("kotsadmdata") because fsgroup is what allows the Kubelet to change the ownership of that volume to be owned by the pod.
		// now, we could just use user "kotsadm" and uid 1001, but since the kotsadm role/clusterrole can also be pre-created with different permissions
		// (not necessarily wildcare permissions), openshift won't allow the pod/container to run with an id that is outside the allowable uid range.
		psc, err := k8sutil.GetOpenShiftPodSecurityContext(deployOptions.Namespace, deployOptions.StrictSecurityContext)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get openshift pod security context")
		}
		securityContext = psc
	}

	var pullSecrets []corev1.LocalObjectReference
	if s := kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.RegistryConfig); s != nil {
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
			Name: "RQLITE_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-rqlite",
					},
					Key: "password",
				},
			},
		},
		{
			Name: "RQLITE_URI",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-rqlite",
					},
					Key: "uri",
				},
			},
		},
		{
			Name: "POSTGRES_URI", // this is still needed for the migration
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-postgres",
					},
					Key:      "uri",
					Optional: pointer.Bool(true),
				},
			},
		},
		{
			Name:  "POSTGRES_SCHEMA_DIR", // this is needed for the migration
			Value: "/scripts/postgres/tables",
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
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
			Name:  "API_ENDPOINT",
			Value: fmt.Sprintf("http://kotsadm.%s.svc.cluster.local:3000", deployOptions.Namespace),
		},
		{
			Name:  "API_ADVERTISE_ENDPOINT",
			Value: "http://localhost:8800",
		},
	}

	env = append(env, GetProxyEnv(deployOptions)...)

	if d := os.Getenv("DISABLE_KOTSADM_OUTBOUND_CONNECTIONS"); d != "" { // used for e2e testing
		env = append(env, corev1.EnvVar{
			Name:  "DISABLE_OUTBOUND_CONNECTIONS",
			Value: d,
		})
	} else if deployOptions.RegistryConfig.OverrideRegistry != "" || deployOptions.Airgap {
		env = append(env, corev1.EnvVar{
			Name:  "DISABLE_OUTBOUND_CONNECTIONS",
			Value: "true",
		})
	}

	if deployOptions.InstallID != "" {
		env = append(env, corev1.EnvVar{
			Name:  "KOTS_INSTALL_ID",
			Value: deployOptions.InstallID,
		})
	}

	if deployOptions.SimultaneousUploads > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "AIRGAP_UPLOAD_PARALLELISM",
			Value: fmt.Sprintf("%d", deployOptions.SimultaneousUploads),
		})
	}

	if deployOptions.PrivateCAsConfigmap != "" {
		env = append(env, corev1.EnvVar{
			Name:  "SSL_CERT_DIR",
			Value: "/certs",
		})
		env = append(env, corev1.EnvVar{
			Name:  "SSL_CERT_CONFIGMAP",
			Value: deployOptions.PrivateCAsConfigmap,
		})
	}

	var storageClassName *string
	if deployOptions.StorageClassName != "" {
		storageClassName = &deployOptions.StorageClassName
	}

	podAnnotations := map[string]string{
		"backup.velero.io/backup-volumes":   "backup",
		"pre.hook.backup.velero.io/command": `["/scripts/backup.sh"]`,
		"pre.hook.backup.velero.io/timeout": "10m",
	}
	for k, v := range deployOptions.AdditionalAnnotations {
		podAnnotations[k] = v
	}
	podLabels := map[string]string{
		"app": "kotsadm",
	}
	for k, v := range deployOptions.AdditionalLabels {
		podLabels[k] = v
	}

	volumes := []corev1.Volume{
		{
			Name: "kotsadmdata",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "kotsadmdata",
				},
			},
		},
		{
			Name: "migrations",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
		{
			Name: "backup",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	if deployOptions.PrivateCAsConfigmap != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "kotsadm-private-cas",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: deployOptions.PrivateCAsConfigmap,
					},
				},
			},
		})
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "kotsadmdata",
			MountPath: "/kotsadmdata",
		},
		{
			Name:      "backup",
			MountPath: "/backup",
		},
		{
			Name:      "tmp",
			MountPath: "/tmp",
		},
	}

	if deployOptions.PrivateCAsConfigmap != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "kotsadm-private-cas",
			MountPath: "/certs",
		})
	}

	statefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "kotsadm",
			Namespace:   deployOptions.Namespace,
			Annotations: deployOptions.AdditionalAnnotations,
			Labels:      types.GetKotsadmLabels(deployOptions.AdditionalLabels),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "kotsadm",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      types.GetKotsadmLabels(podLabels),
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: defaultKOTSNodeAffinity(),
					},
					Tolerations:        deployOptions.Tolerations,
					SecurityContext:    securityContext,
					Volumes:            volumes,
					ServiceAccountName: "kotsadm",
					RestartPolicy:      corev1.RestartPolicyAlways,
					ImagePullSecrets:   pullSecrets,
					InitContainers: []corev1.Container{
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm-migrations"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "schemahero-plan",
							Args:            []string{"plan"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "migrations",
									MountPath: "/migrations",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "SCHEMAHERO_DRIVER",
									Value: "rqlite",
								},
								{
									Name:  "SCHEMAHERO_SPEC_FILE",
									Value: "/tables",
								},
								{
									Name:  "SCHEMAHERO_OUT",
									Value: "/migrations/plan.yaml",
								},
								{
									Name: "SCHEMAHERO_URI",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-rqlite",
											},
											Key: "uri",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("50m"),
									"memory": resource.MustParse("50Mi"),
								},
							},
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
						},
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm-migrations"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "schemahero-apply",
							Args:            []string{"apply"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "migrations",
									MountPath: "/migrations",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "SCHEMAHERO_DRIVER",
									Value: "rqlite",
								},
								{
									Name:  "SCHEMAHERO_DDL",
									Value: "/migrations/plan.yaml",
								},
								{
									Name: "SCHEMAHERO_URI",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-rqlite",
											},
											Key: "uri",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("50m"),
									"memory": resource.MustParse("50Mi"),
								},
							},
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
						},
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "restore-data",
							Command: []string{
								"/scripts/restore.sh",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kotsadmdata",
									MountPath: "/kotsadmdata",
								},
								{
									Name:      "backup",
									MountPath: "/backup",
								},
								{
									Name:      "tmp",
									MountPath: "/tmp",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "RQLITE_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-rqlite",
											},
											Key: "password",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("1"),
									"memory": resource.MustParse("2Gi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
							},
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
						},
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "migrate-s3",
							Command: []string{
								"/scripts/migrate-s3.sh",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kotsadmdata",
									MountPath: "/kotsadmdata",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "S3_ENDPOINT",
									Value: fmt.Sprintf("http://kotsadm-minio.%s.svc.cluster.local:9000", deployOptions.Namespace),
								},
								{
									Name:  "S3_BUCKET_NAME",
									Value: "kotsadm",
								},
								{
									Name: "S3_ACCESS_KEY_ID",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-minio",
											},
											Key:      "accesskey",
											Optional: pointer.Bool(true),
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
											Key:      "secretkey",
											Optional: pointer.Bool(true),
										},
									},
								},
								{
									Name:  "S3_BUCKET_ENDPOINT",
									Value: "true",
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("1"),
									"memory": resource.MustParse("2Gi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
							},
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
						},
					},
					Containers: []corev1.Container{
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "kotsadm",
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
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(3000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							VolumeMounts: volumeMounts,
							Env:          env,
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("1"),
									"memory": resource.MustParse("2Gi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
							},
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "kotsadmdata",
						Labels: types.GetKotsadmLabels(),
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						StorageClassName: storageClassName,
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName(corev1.ResourceStorage): size,
							},
						},
					},
				},
			},
		},
	}

	return statefulset, nil
}

func KotsadmService(namespace string, nodePort int32) *corev1.Service {
	port := corev1.ServicePort{
		Name:       "http",
		Port:       3000,
		TargetPort: intstr.FromString("http"),
		NodePort:   nodePort,
	}

	serviceType := corev1.ServiceTypeClusterIP
	if nodePort != 0 {
		serviceType = corev1.ServiceTypeNodePort
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm",
			},
			Type: serviceType,
			Ports: []corev1.ServicePort{
				port,
			},
		},
	}

	return service
}

func KotsadmIngress(namespace string, ingressConfig kotsv1beta1.IngressResourceConfig) *networkingv1.Ingress {
	return ingress.IngressFromConfig(namespace, ingressConfig, "kotsadm", "kotsadm", 3000, nil)
}
