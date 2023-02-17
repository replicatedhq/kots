package kotsadm

import (
	_ "embed"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

var (
	//go:embed scripts/migrate-minio-xl.sh
	migrateMinioXlScript string
)

func MinioStatefulset(deployOptions types.DeployOptions, size resource.Quantity, name string) (*appsv1.StatefulSet, error) {
	image := GetAdminConsoleImage(deployOptions, "minio")

	var pullSecrets []corev1.LocalObjectReference
	if s := kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.RegistryConfig); s != nil {
		pullSecrets = []corev1.LocalObjectReference{
			{
				Name: s.ObjectMeta.Name,
			},
		}
	}

	securityContext := securePodContext(1001, deployOptions.StrictSecurityContext)
	if deployOptions.IsOpenShift {
		// need to use a security context here because if the project is running with a scc that has "MustRunAsNonRoot" (or is not "MustRunAsRange"),
		// openshift won't assign a user id to the container to run with, and the container will try to run as root and fail.
		psc, err := k8sutil.GetOpenShiftPodSecurityContext(deployOptions.Namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get openshift pod security context")
		}
		securityContext = psc
	}

	statefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   name,
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
						"app": name,
					}),
					Annotations: map[string]string{
						"backup.velero.io/backup-volumes": "kotsadm-minio,minio-config-dir,minio-cert-dir",
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext:  securityContext,
					ImagePullSecrets: pullSecrets,
					Volumes: []corev1.Volume{
						{
							Name: name,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: name,
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
					},
					Containers: []corev1.Container{
						{
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            name,
							Command: []string{
								"/bin/sh",
								"-ce",
								"/usr/bin/docker-entrypoint.sh minio -C /home/minio/.minio/ --quiet server /export",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "service",
									ContainerPort: 9000,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      name,
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
							},
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
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("512Mi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("50m"),
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

func MinioService(namespace string, name string) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
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

func MinioXlMigrationScriptsConfigMap(namespace string) *corev1.ConfigMap {
	data := map[string]string{}

	data["migrate-minio-xl.sh"] = migrateMinioXlScript

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio-xl-migration-scripts",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Data: data,
	}
	return configMap
}

func MinioXlMigrationJob(deployOptions types.DeployOptions, namespace string) *batchv1.Job {
	scriptsFileMode := int32(0755)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-migrate-minio-xl",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "kotsadm-migrate-minio-xl",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "kotsadm-migrate-minio-xl",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Image:           GetAdminConsoleImage(deployOptions, "minio-client"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "migrate-minio-xl",
							Command: []string{
								"/scripts/migrate-minio-xl.sh",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "S3_ENDPOINT_OLD",
									Value: "http://kotsadm-minio:9000",
								},
								{
									Name:  "S3_ENDPOINT_NEW",
									Value: "http://kotsadm-minio-xl-migration:9000",
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
											Optional: pointer.BoolPtr(true),
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
											Optional: pointer.BoolPtr(true),
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
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("512Mi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("50m"),
									"memory": resource.MustParse("100Mi"),
								},
							},
							SecurityContext: secureContainerContext(deployOptions.StrictSecurityContext),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kotsadm-minio-xl-migration-scripts",
									MountPath: "/scripts",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "kotsadm-minio-xl-migration-scripts",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "kotsadm-minio-xl-migration-scripts",
									},
									DefaultMode: &scriptsFileMode,
									Items: []corev1.KeyToPath{
										{
											Key:  "migrate-minio-xl.sh",
											Path: "migrate-minio-xl.sh",
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
}
