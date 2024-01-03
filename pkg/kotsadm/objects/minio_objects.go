package kotsadm

import (
	_ "embed"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	MinioXlMigrationScriptsConfigmapName = "kotsadm-minio-xl-migration-scripts"
)

var (
	//go:embed scripts/copy-minio-client.sh
	copyMinioClient string
	//go:embed scripts/export-minio-data.sh
	exportMinioData string
	//go:embed scripts/import-minio-data.sh
	importMinioData string
)

func MinioStatefulset(deployOptions types.DeployOptions, size resource.Quantity) (*appsv1.StatefulSet, error) {
	image := GetAdminConsoleImage(deployOptions, "minio")

	var pullSecrets []corev1.LocalObjectReference
	if s := kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.RegistryConfig); s != nil {
		pullSecrets = []corev1.LocalObjectReference{
			{
				Name: s.ObjectMeta.Name,
			},
		}
	}

	securityContext := k8sutil.SecurePodContext(1001, 1001, deployOptions.StrictSecurityContext)
	if deployOptions.IsOpenShift {
		// need to use a security context here because if the project is running with a scc that has "MustRunAsNonRoot" (or is not "MustRunAsRange"),
		// openshift won't assign a user id to the container to run with, and the container will try to run as root and fail.
		psc, err := k8sutil.GetOpenShiftPodSecurityContext(deployOptions.Namespace, deployOptions.StrictSecurityContext)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get openshift pod security context")
		}
		securityContext = psc
	}

	cpuRequest, cpuLimit := "50m", "100m"
	memoryRequest, memoryLimit := "100Mi", "512Mi"

	if deployOptions.IsGKEAutopilot {
		// requests and limits must be the same for GKE autopilot: https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-resource-requests#resource-limits
		// otherwise, the limit will be lowered to match the request
		// additionally, cpu requests must be in multiples of 250m: https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-resource-requests#min-max-requests
		cpuRequest, cpuLimit = "250m", "250m"
		memoryRequest, memoryLimit = "512Mi", "512Mi"
	}

	resourceRequirements := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse(cpuLimit),
			"memory": resource.MustParse(memoryLimit),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse(cpuRequest),
			"memory": resource.MustParse(memoryRequest),
		},
	}

	initContainers := []corev1.Container{}
	if deployOptions.MigrateToMinioXl {
		initContainers = append(initContainers, migrateToMinioXlInitContainers(deployOptions, resourceRequirements)...)
	}

	statefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-minio",
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "kotsadm-minio",
						Labels: types.GetKotsadmLabels(),
					},
					Spec: corev1.PersistentVolumeClaimSpec{
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
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: types.GetKotsadmLabels(map[string]string{
						"app": "kotsadm-minio",
					}),
					Annotations: map[string]string{
						"backup.velero.io/backup-volumes": "kotsadm-minio,minio-config-dir,minio-cert-dir",
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext:  securityContext,
					ImagePullSecrets: pullSecrets,
					InitContainers:   initContainers,
					Volumes:          minioVolumes(deployOptions),
					Containers: []corev1.Container{
						{
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "kotsadm-minio",
							Command: []string{
								"/bin/sh",
								"-ce",
								"minio -C /home/minio/.minio/ --quiet server /export",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "service",
									ContainerPort: 9000,
								},
							},
							VolumeMounts: minioVolumeMounts(),
							Env: []corev1.EnvVar{
								{
									Name: "MINIO_ACCESS_KEY",
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
									Name: "MINIO_SECRET_KEY",
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
									Name:  "MINIO_BROWSER",
									Value: "on",
								},
								{
									Name:  "MINIO_UPDATE",
									Value: "off",
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								FailureThreshold:    3,
								SuccessThreshold:    1,
								PeriodSeconds:       30,
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/minio/health/live",
										Port:   intstr.FromString("service"),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								FailureThreshold:    3,
								SuccessThreshold:    1,
								PeriodSeconds:       15,
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/minio/health/ready",
										Port:   intstr.FromString("service"),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							Resources:       resourceRequirements,
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
						},
					},
				},
			},
		},
	}

	return statefulset, nil
}

func migrateToMinioXlInitContainers(deployOptions types.DeployOptions, resourceRequirements corev1.ResourceRequirements) []corev1.Container {
	volumeMounts := append(minioVolumeMounts(), minioXlMigrationVolumeMounts()...)

	return []corev1.Container{
		{
			Image:           GetAdminConsoleImage(deployOptions, "minio"),
			ImagePullPolicy: corev1.PullIfNotPresent,
			Name:            "copy-minio-client",
			Command: []string{
				"/scripts/copy-minio-client.sh",
			},
			VolumeMounts: volumeMounts,
			Env: []corev1.EnvVar{
				{
					Name:  "KOTSADM_MINIO_MIGRATION_DIR",
					Value: "/migration",
				},
			},
			Resources:       resourceRequirements,
			SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
		},
		{
			Image:           deployOptions.CurrentMinioImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Name:            "export-minio-data",
			Command: []string{
				"/scripts/export-minio-data.sh",
			},
			VolumeMounts: volumeMounts,
			Env: []corev1.EnvVar{
				{
					Name:  "KOTSADM_MINIO_ENDPOINT",
					Value: "http://localhost:9000",
				},
				{
					Name:  "KOTSADM_MINIO_BUCKET_NAME",
					Value: "kotsadm",
				},
				{
					Name: "MINIO_ACCESS_KEY",
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
					Name: "MINIO_SECRET_KEY",
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
					Name:  "KOTSADM_MINIO_LEGACY_ALIAS",
					Value: "legacy",
				},
				{
					Name:  "KOTSADM_MINIO_MIGRATION_DIR",
					Value: "/migration",
				},
			},
			Resources:       resourceRequirements,
			SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
		},
		{
			Image:           GetAdminConsoleImage(deployOptions, "minio"),
			ImagePullPolicy: corev1.PullIfNotPresent,
			Name:            "import-minio-data",
			Command: []string{
				"/scripts/import-minio-data.sh",
			},
			VolumeMounts: volumeMounts,
			Env: []corev1.EnvVar{
				{
					Name:  "KOTSADM_MINIO_ENDPOINT",
					Value: "http://localhost:9000",
				},
				{
					Name:  "KOTSADM_MINIO_BUCKET_NAME",
					Value: "kotsadm",
				},
				{
					Name: "MINIO_ACCESS_KEY",
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
					Name: "MINIO_SECRET_KEY",
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
					Name:  "KOTSADM_MINIO_NEW_ALIAS",
					Value: "new",
				},
				{
					Name:  "KOTSADM_MINIO_MIGRATION_DIR",
					Value: "/migration",
				},
			},
			Resources:       resourceRequirements,
			SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
		},
	}
}

func minioVolumes(deployOptions types.DeployOptions) []corev1.Volume {
	scriptsFileMode := int32(0755)

	volumes := []corev1.Volume{
		{
			Name: "kotsadm-minio",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "kotsadm-minio",
				},
			},
		},
		{
			Name: "minio-config-dir",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "minio-cert-dir",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	if deployOptions.MigrateToMinioXl {
		volumes = append(volumes, corev1.Volume{
			Name: "kotsadm-minio-client-config",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}, corev1.Volume{
			Name: "kotsadm-minio-xl-migration",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}, corev1.Volume{
			Name: "kotsadm-minio-xl-migration-scripts",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: MinioXlMigrationScriptsConfigmapName,
					},
					DefaultMode: &scriptsFileMode,
					Items: []corev1.KeyToPath{
						{
							Key:  "copy-minio-client.sh",
							Path: "copy-minio-client.sh",
						},
						{
							Key:  "export-minio-data.sh",
							Path: "export-minio-data.sh",
						},
						{
							Key:  "import-minio-data.sh",
							Path: "import-minio-data.sh",
						},
					},
				},
			},
		})
	}

	return volumes
}

func minioVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "kotsadm-minio",
			MountPath: "/export",
		},
		{
			Name:      "minio-config-dir",
			MountPath: "/home/minio/.minio/",
		},
		{
			Name:      "minio-cert-dir",
			MountPath: "/.minio/",
		},
	}
}

func minioXlMigrationVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "kotsadm-minio-client-config",
			MountPath: "/.mc",
		},
		{
			Name:      "kotsadm-minio-xl-migration",
			MountPath: "/migration",
		},
		{
			Name:      "kotsadm-minio-xl-migration-scripts",
			MountPath: "/scripts",
		},
	}
}

func MinioXlMigrationScriptsConfigMap(namespace string) *corev1.ConfigMap {
	data := map[string]string{}

	data["copy-minio-client.sh"] = copyMinioClient
	data["export-minio-data.sh"] = exportMinioData
	data["import-minio-data.sh"] = importMinioData

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      MinioXlMigrationScriptsConfigmapName,
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Data: data,
	}
	return configMap
}

func MinioService(namespace string) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm-minio",
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "service",
					Port:       9000,
					TargetPort: intstr.FromInt(9000),
				},
			},
		},
	}

	return service
}
