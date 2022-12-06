package kotsadm

import (
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

func RqliteStatefulset(deployOptions types.DeployOptions, size resource.Quantity) (*appsv1.StatefulSet, error) {
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

	volumes := getRqliteVolumes()
	volumeMounts := getRqliteVolumeMounts()

	cpuRequest, cpuLimit := "100m", "200m"
	memoryRequest, memoryLimit := "100Mi", "200Mi"

	if deployOptions.IsGKEAutopilot {
		// need to increase the cpu and memory request to meet GKE Autopilot's minimum requirement of 500m when using pod anti affinity
		cpuRequest, cpuLimit = "500m", "500m"
		memoryRequest, memoryLimit = "512Mi", "512Mi"
	}

	statefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-rqlite",
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName:         "kotsadm-rqlite-headless",
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-rqlite",
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "kotsadm-rqlite",
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
						"app": "kotsadm-rqlite",
					}),
				},
				Spec: corev1.PodSpec{
					SecurityContext:  securityContext,
					ImagePullSecrets: pullSecrets,
					Volumes:          volumes,
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app",
												Operator: metav1.LabelSelectorOpIn,
												Values: []string{
													"kotsadm-rqlite",
												},
											},
										},
									},
									TopologyKey: corev1.LabelHostname,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Image:           GetAdminConsoleImage(deployOptions, "rqlite"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "kotsadm-rqlite",
							Args: []string{
								"-disco-mode=dns",
								"-disco-config={\"name\":\"kotsadm-rqlite-headless\"}",
								"-bootstrap-expect=1",
								"-auth=/auth/config.json",
								"-join-as=kotsadm",
								"-on-disk",
								"-on-disk-startup",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "rqlite",
									ContainerPort: 4001,
								},
							},
							VolumeMounts: volumeMounts,
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
								FailureThreshold:    3,
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/readyz?noleader",
										Port:   intstr.FromString("rqlite"),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: 1,
								PeriodSeconds:       1,
								TimeoutSeconds:      5,
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/readyz",
										Port:   intstr.FromString("rqlite"),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse(cpuLimit),
									"memory": resource.MustParse(memoryLimit),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse(cpuRequest),
									"memory": resource.MustParse(memoryRequest),
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

func RqliteService(namespace string) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-rqlite",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm-rqlite",
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "rqlite",
					Port:       4001,
					TargetPort: intstr.FromString("rqlite"),
				},
			},
		},
	}

	return service
}

func RqliteHeadlessService(namespace string) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-rqlite-headless",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm-rqlite",
			},
			Type:                     corev1.ServiceTypeClusterIP,
			ClusterIP:                corev1.ClusterIPNone,
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:       "rqlite",
					Port:       4001,
					TargetPort: intstr.FromString("rqlite"),
				},
			},
		},
	}

	return service
}

func getRqliteVolumes() []corev1.Volume {
	scriptsFileMode := int32(0755)

	volumes := []corev1.Volume{
		{
			Name: "tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "authconfig",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  "kotsadm-rqlite",
					DefaultMode: &scriptsFileMode,
					Items: []corev1.KeyToPath{
						{
							Key:  "authconfig.json",
							Path: "authconfig.json",
						},
					},
				},
			},
		},
	}

	return volumes
}

func getRqliteVolumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "kotsadm-rqlite",
			MountPath: "/rqlite/file",
		},
		{
			Name:      "tmp",
			MountPath: "/tmp",
		},
		{
			Name:      "authconfig",
			MountPath: "/auth/config.json",
			SubPath:   "authconfig.json",
		},
	}

	return volumeMounts
}
