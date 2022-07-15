package kotsadm

import (
	"fmt"

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
	POSTGRES_USER             = "kotsadm"
	POSTGRES_DB               = "kotsadm"
	POSTGRES_HOST_AUTH_METHOD = "md5"
	POSTGRES_UPGRADE_PORT     = "50432" // run on different port to avoid unintended client connections.
	POSTGRES_UPGRADE_DIR      = "/var/lib/postgresql/upgrade"
	POSTGRES_PVC_MOUNT_PATH   = "/var/lib/postgresql/data"
	POSTGRES_10_DATA_DIR      = "/var/lib/postgresql/data/pgdata"
	POSTGRES_14_DATA_DIR      = "/var/lib/postgresql/data/pg14data"
)

func PostgresStatefulset(deployOptions types.DeployOptions, size resource.Quantity) (*appsv1.StatefulSet, error) {
	var pullSecrets []corev1.LocalObjectReference
	if s := kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.RegistryConfig); s != nil {
		pullSecrets = []corev1.LocalObjectReference{
			{
				Name: s.ObjectMeta.Name,
			},
		}
	}

	securityContext := securePodContext(999, deployOptions.StrictSecurityContext)
	if deployOptions.IsOpenShift {
		// need to use a security context here because if the project is running with a scc that has "MustRunAsNonRoot" (or is not "MustRunAsRange"),
		// openshift won't assign a user id to the container to run with, and the container will try to run as root and fail.
		psc, err := k8sutil.GetOpenShiftPodSecurityContext(deployOptions.Namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get openshift pod security context")
		}
		securityContext = psc
	}

	volumes := getPostgresVolumes(deployOptions)
	volumeMounts := getPostgresVolumeMounts(deployOptions)

	initContainers := []corev1.Container{}
	initContainers = append(initContainers, getPostgresUpgradeInitContainers(deployOptions, volumeMounts)...)

	statefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-postgres",
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-postgres",
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "kotsadm-postgres",
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
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: types.GetKotsadmLabels(map[string]string{
						"app": "kotsadm-postgres",
					}),
				},
				Spec: corev1.PodSpec{
					SecurityContext:  securityContext,
					ImagePullSecrets: pullSecrets,
					Volumes:          volumes,
					InitContainers:   initContainers,
					Containers: []corev1.Container{
						{
							Image:           GetAdminConsoleImage(deployOptions, "postgres-14"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "kotsadm-postgres",
							Ports: []corev1.ContainerPort{
								{
									Name:          "postgres",
									ContainerPort: 5432,
								},
							},
							VolumeMounts: volumeMounts,
							Env: []corev1.EnvVar{
								{
									Name:  "PGDATA",
									Value: POSTGRES_14_DATA_DIR,
								},
								{
									Name:  "POSTGRES_USER",
									Value: POSTGRES_USER,
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
									Name:  "POSTGRES_DB",
									Value: POSTGRES_DB,
								},
								{
									Name:  "POSTGRES_HOST_AUTH_METHOD",
									Value: POSTGRES_HOST_AUTH_METHOD,
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
								FailureThreshold:    3,
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/sh",
											"-i",
											"-c",
											"pg_isready -U kotsadm -h 127.0.0.1 -p 5432",
										},
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: 1,
								PeriodSeconds:       1,
								TimeoutSeconds:      5,
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/sh",
											"-i",
											"-c",
											"pg_isready -U kotsadm -h 127.0.0.1 -p 5432",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("200m"),
									"memory": resource.MustParse("200Mi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
							},
							SecurityContext: secureContainerContext(deployOptions.StrictSecurityContext),
						},
					},
				},
			},
		},
	}

	return statefulset, nil
}

func PostgresService(namespace string) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-postgres",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm-postgres",
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Port:       5432,
					TargetPort: intstr.FromString("postgres"),
				},
			},
		},
	}

	return service
}

func getPostgresVolumes(deployOptions types.DeployOptions) []corev1.Volume {
	scriptsFileMode := int32(0755)

	volumes := []corev1.Volume{
		{
			Name: "kotsadm-postgres",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "kotsadm-postgres",
				},
			},
		},
		{
			Name: "upgrade",
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
		{
			Name: "run",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "scripts",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-postgres",
					},
					DefaultMode: &scriptsFileMode,
					Items: []corev1.KeyToPath{
						{
							Key:  "copy-postgres-10.sh",
							Path: "copy-postgres-10.sh",
						},
						{
							Key:  "upgrade-postgres.sh",
							Path: "upgrade-postgres.sh",
						},
					},
				},
			},
		},
	}

	if !deployOptions.IsOpenShift {
		// this is needed for user remapping because older versions used to run with a different uid
		passwdFileMode := int32(0644)
		volumes = append(volumes, corev1.Volume{
			Name: "etc-passwd",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "kotsadm-postgres",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "passwd",
							Path: "passwd",
							Mode: &passwdFileMode,
						},
					},
				},
			},
		})
	}

	return volumes
}

func getPostgresVolumeMounts(deployOptions types.DeployOptions) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "kotsadm-postgres",
			MountPath: POSTGRES_PVC_MOUNT_PATH,
		},
		{
			Name:      "upgrade",
			MountPath: POSTGRES_UPGRADE_DIR,
		},
		{
			Name:      "tmp",
			MountPath: "/tmp",
		},
		{
			Name:      "run",
			MountPath: "/var/run/postgresql",
		},
		{
			Name:      "scripts",
			MountPath: "/scripts",
		},
	}

	if !deployOptions.IsOpenShift {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "etc-passwd",
			MountPath: "/etc/passwd",
			SubPath:   "passwd",
		})
	}

	return volumeMounts
}

func getPostgresUpgradeInitContainers(deployOptions types.DeployOptions, volumeMounts []corev1.VolumeMount) []corev1.Container {
	return []corev1.Container{
		{
			Image:           GetAdminConsoleImage(deployOptions, "postgres-10"),
			ImagePullPolicy: corev1.PullIfNotPresent,
			Name:            "copy-postgres-10",
			Command: []string{
				"/scripts/copy-postgres-10.sh",
			},
			VolumeMounts: volumeMounts,
			Env: []corev1.EnvVar{
				{
					Name:  "PGDATA",
					Value: POSTGRES_10_DATA_DIR,
				},
				{
					Name:  "POSTGRES_UPGRADE_DIR",
					Value: POSTGRES_UPGRADE_DIR,
				},
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("200m"),
					"memory": resource.MustParse("200Mi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("100m"),
					"memory": resource.MustParse("100Mi"),
				},
			},
			SecurityContext: secureContainerContext(deployOptions.StrictSecurityContext),
		},
		{
			Image:           GetAdminConsoleImage(deployOptions, "postgres-14"),
			ImagePullPolicy: corev1.PullIfNotPresent,
			Name:            "upgrade-postgres",
			Command: []string{
				"/scripts/upgrade-postgres.sh",
			},
			VolumeMounts: volumeMounts,
			Env: []corev1.EnvVar{
				{
					Name:  "PGPORT",
					Value: POSTGRES_UPGRADE_PORT,
				},
				{
					Name:  "PGDATA",
					Value: POSTGRES_14_DATA_DIR,
				},
				{
					Name:  "POSTGRES_USER",
					Value: POSTGRES_USER,
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
					Name:  "POSTGRES_DB",
					Value: POSTGRES_DB,
				},
				{
					Name:  "POSTGRES_HOST_AUTH_METHOD",
					Value: POSTGRES_HOST_AUTH_METHOD,
				},
				{
					Name:  "POSTGRES_UPGRADE_DIR",
					Value: POSTGRES_UPGRADE_DIR,
				},
				{
					Name:  "PGDATAOLD",
					Value: POSTGRES_10_DATA_DIR,
				},
				{
					Name:  "PGDATANEW",
					Value: POSTGRES_14_DATA_DIR,
				},
				{
					Name:  "PGBINOLD",
					Value: fmt.Sprintf("%s/pg10/bin", POSTGRES_UPGRADE_DIR),
				},
				{
					Name:  "PGBINNEW",
					Value: "/usr/local/bin",
				},
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("200m"),
					"memory": resource.MustParse("200Mi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("100m"),
					"memory": resource.MustParse("100Mi"),
				},
			},
			SecurityContext: secureContainerContext(deployOptions.StrictSecurityContext),
		},
	}
}
