package kotsadm

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	"github.com/replicatedhq/kots/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

	// image
	existingDeployment.Spec.Template.Spec.Containers[containerIdx].Image = desiredDeployment.Spec.Template.Spec.Containers[0].Image

	additionalInitContainers := []corev1.Container{}
	for _, desiredContainer := range desiredDeployment.Spec.Template.Spec.InitContainers {
		found := false
		for i, existingContainer := range existingDeployment.Spec.Template.Spec.InitContainers {
			if existingContainer.Name != desiredContainer.Name {
				continue
			}

			existingDeployment.Spec.Template.Spec.InitContainers[i] = *desiredContainer.DeepCopy()
			found = true
			break
		}
		if !found {
			additionalInitContainers = append(additionalInitContainers, *desiredContainer.DeepCopy())
		}
	}
	existingDeployment.Spec.Template.Spec.InitContainers = append(existingDeployment.Spec.Template.Spec.InitContainers, additionalInitContainers...)

	newVolumes := []corev1.Volume{}
	for _, v := range desiredDeployment.Spec.Template.Spec.Volumes {
		newVolumes = append(newVolumes, *v.DeepCopy())
	}
	existingDeployment.Spec.Template.Spec.Volumes = newVolumes

	// copy the env vars from the desired to existing. this could undo a change that the user had.
	// we don't know which env vars we set and which are user edited. this method avoids deleting
	// env vars that the user added, but doesn't handle edited vars
	mergedEnvs := []corev1.EnvVar{}
	for _, env := range desiredDeployment.Spec.Template.Spec.Containers[0].Env {
		mergedEnvs = append(mergedEnvs, env)
	}
	for _, existingEnv := range existingDeployment.Spec.Template.Spec.Containers[containerIdx].Env {
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
	existingDeployment.Spec.Template.Spec.Containers[containerIdx].Env = mergedEnvs

	return nil
}

func KotsadmDeployment(deployOptions types.DeployOptions) (*appsv1.Deployment, error) {
	securityContext := &corev1.PodSecurityContext{
		RunAsUser: util.IntPointer(1001),
		FSGroup:   util.IntPointer(1001),
	}
	if deployOptions.IsOpenShift {
		psc, err := k8sutil.GetOpenShiftPodSecurityContext(deployOptions.Namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get openshift pod security context")
		}
		securityContext = psc
	}

	var pullSecrets []corev1.LocalObjectReference
	if s := kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.KotsadmOptions); s != nil {
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
			Name: "POSTGRES_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-postgres",
					},
					Key: "password",
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
	}
	if strings.HasPrefix(deployOptions.StorageBaseURI, "docker://") {
		env = append(env, corev1.EnvVar{
			Name:  "STORAGE_BASEURI",
			Value: deployOptions.StorageBaseURI,
		})
		env = append(env, corev1.EnvVar{
			Name:  "STORAGE_BASEURI_PLAINHTTP",
			Value: strconv.FormatBool(deployOptions.StorageBaseURIPlainHTTP),
		})
	} else {
		s3env := []corev1.EnvVar{
			{
				Name:  "S3_ENDPOINT",
				Value: "http://kotsadm-minio:9000",
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
		env = append(env, s3env...)
	}

	env = append(env, GetProxyEnv(deployOptions)...)
	if deployOptions.KotsadmOptions.OverrideRegistry != "" || deployOptions.Airgap {
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

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm",
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: types.GetKotsadmLabels(map[string]string{
						"app": "kotsadm",
					}),
					Annotations: map[string]string{
						"backup.velero.io/backup-volumes":   "backup",
						"pre.hook.backup.velero.io/command": `["/backup.sh"]`,
						"pre.hook.backup.velero.io/timeout": "10m",
					},
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: defaultKotsNodeAffinity(),
					},
					SecurityContext: securityContext,
					Volumes: []corev1.Volume{
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
					},
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
									Value: "postgres",
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
												Name: "kotsadm-postgres",
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
									Value: "postgres",
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
												Name: "kotsadm-postgres",
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
						},
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "restore-db",
							Command: []string{
								"/restore-db.sh",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "backup",
									MountPath: "/backup",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POSTGRES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-postgres",
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
						},
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "restore-s3",
							Command: []string{
								"/restore-s3.sh",
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
									Value: "http://kotsadm-minio:9000",
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
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(3000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "backup",
									MountPath: "/backup",
								},
							},
							Env: env,
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

	// image
	existingStatefulset.Spec.Template.Spec.Containers[containerIdx].Image = desiredStatefulSet.Spec.Template.Spec.Containers[0].Image

	additionalInitContainers := []corev1.Container{}
	for _, desiredContainer := range desiredStatefulSet.Spec.Template.Spec.InitContainers {
		found := false
		for i, existingContainer := range existingStatefulset.Spec.Template.Spec.InitContainers {
			if existingContainer.Name != desiredContainer.Name {
				continue
			}

			existingStatefulset.Spec.Template.Spec.InitContainers[i] = *desiredContainer.DeepCopy()
			found = true
			break
		}

		if !found {
			additionalInitContainers = append(additionalInitContainers, *desiredContainer.DeepCopy())
		}
	}
	existingStatefulset.Spec.Template.Spec.InitContainers = append(existingStatefulset.Spec.Template.Spec.InitContainers, additionalInitContainers...)

	newVolumes := []corev1.Volume{}
	for _, v := range desiredStatefulSet.Spec.Template.Spec.Volumes {
		newVolumes = append(newVolumes, *v.DeepCopy())
	}
	existingStatefulset.Spec.Template.Spec.Volumes = newVolumes

	// copy the env vars from the desired to existing. this could undo a change that the user had.
	// we don't know which env vars we set and which are user edited. this method avoids deleting
	// env vars that the user added, but doesn't handle edited vars
	mergedEnvs := []corev1.EnvVar{}
	for _, env := range desiredStatefulSet.Spec.Template.Spec.Containers[0].Env {
		mergedEnvs = append(mergedEnvs, env)
	}
	for _, existingEnv := range existingStatefulset.Spec.Template.Spec.Containers[containerIdx].Env {
		isUnxpected := true
		for _, env := range desiredStatefulSet.Spec.Template.Spec.Containers[0].Env {
			if env.Name == existingEnv.Name {
				isUnxpected = false
			}
		}

		if isUnxpected {
			mergedEnvs = append(mergedEnvs, existingEnv)
		}
	}
	existingStatefulset.Spec.Template.Spec.Containers[containerIdx].Env = mergedEnvs

	return nil
}

func KotsadmStatefulSet(deployOptions types.DeployOptions, size resource.Quantity) (*appsv1.StatefulSet, error) {
	securityContext := &corev1.PodSecurityContext{
		RunAsUser: util.IntPointer(1001),
		FSGroup:   util.IntPointer(1001),
	}
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
		psc, err := k8sutil.GetOpenShiftPodSecurityContext(deployOptions.Namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get openshift pod security context")
		}
		securityContext = psc
	}

	var pullSecrets []corev1.LocalObjectReference
	if s := kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.KotsadmOptions); s != nil {
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
			Name: "POSTGRES_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-postgres",
					},
					Key: "password",
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

	if strings.HasPrefix(deployOptions.StorageBaseURI, "docker://") {
		env = append(env, corev1.EnvVar{
			Name:  "STORAGE_BASEURI",
			Value: deployOptions.StorageBaseURI,
		})
		env = append(env, corev1.EnvVar{
			Name:  "STORAGE_BASEURI_PLAINHTTP",
			Value: strconv.FormatBool(deployOptions.StorageBaseURIPlainHTTP),
		})
	}

	env = append(env, GetProxyEnv(deployOptions)...)

	if deployOptions.KotsadmOptions.OverrideRegistry != "" || deployOptions.Airgap {
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

	trueVal := true

	statefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm",
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
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
					Labels: types.GetKotsadmLabels(map[string]string{
						"app": "kotsadm",
					}),
					Annotations: map[string]string{
						"backup.velero.io/backup-volumes":   "backup",
						"pre.hook.backup.velero.io/command": `["/backup.sh"]`,
						"pre.hook.backup.velero.io/timeout": "10m",
					},
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: defaultKotsNodeAffinity(),
					},
					SecurityContext: securityContext,
					Volumes: []corev1.Volume{
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
					},
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
									Value: "postgres",
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
												Name: "kotsadm-postgres",
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
									Value: "postgres",
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
												Name: "kotsadm-postgres",
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
						},
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "restore-data",
							Command: []string{
								"/restore.sh",
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
							},
							Env: []corev1.EnvVar{
								{
									Name: "POSTGRES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-postgres",
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
						},
						{
							Image:           GetAdminConsoleImage(deployOptions, "kotsadm"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "migrate-s3",
							Command: []string{
								"/migrate-s3.sh",
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
									Value: "http://kotsadm-minio:9000",
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
											Optional: &trueVal,
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
											Optional: &trueVal,
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
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(3000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
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
							},
							Env: env,
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
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
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

func KotsadmIngress(namespace string, ingressConfig kotsv1beta1.IngressResourceConfig) *extensionsv1beta1.Ingress {
	return ingress.IngressFromConfig(ingressConfig, "kotsadm", "kotsadm", 3000, nil)
}
