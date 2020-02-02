package kotsadm

import (
	"fmt"
	"github.com/replicatedhq/kots/pkg/kotsadm/hostnetwork"
	"os"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var minioPVSize = resource.MustParse("4Gi")

func minioStatefulset(deployOptions types.DeployOptions) *appsv1.StatefulSet {
	size := minioPVSize

	if deployOptions.LimitRange != nil {
		var allowedMax *resource.Quantity
		var allowedMin *resource.Quantity

		for _, limit := range deployOptions.LimitRange.Spec.Limits {
			if limit.Type == corev1.LimitTypePersistentVolumeClaim {
				max, ok := limit.Max[corev1.ResourceStorage]
				if ok {
					allowedMax = &max
				}

				min, ok := limit.Min[corev1.ResourceStorage]
				if ok {
					allowedMin = &min
				}
			}
		}

		newSize := promptForSizeIfNotBetween("minio", &size, allowedMin, allowedMax)
		if newSize == nil {
			os.Exit(-1)
		}

		size = *newSize
	}

	var securityContext corev1.PodSecurityContext
	var initContainers []corev1.Container
	if !deployOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(1001),
			FSGroup:   util.IntPointer(1001),
		}

		initContainers = []corev1.Container{
			{
				Image:           fmt.Sprintf("%s/minio:%s", kotsadmRegistry(), kotsadmTag()),
				ImagePullPolicy: corev1.PullIfNotPresent,
				Name:            "kotsadm-minio-init",
				Command: []string{
					"/bin/sh",
					"-ce",
					"chown -R minio:minio /export && chown -R minio:minio /home/minio/.minio",
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "kotsadm-minio",
						MountPath: "/export",
					},
					{
						Name:      "minio-config-dir",
						MountPath: "/home/minio/.minio/",
					},
				},
			},
		}
	}

	statefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: deployOptions.Namespace,
			Labels: map[string]string{
				types.KotsadmKey: types.KotsadmLabelValue,
			},
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
						Name: "kotsadm-minio",
						Labels: map[string]string{
							types.KotsadmKey: types.KotsadmLabelValue,
						},
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
					Labels: map[string]string{
						"app":            "kotsadm-minio",
						types.KotsadmKey: types.KotsadmLabelValue,
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &securityContext,
					Volumes: []corev1.Volume{
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
					},
					Tolerations: hostnetwork.Tolerations(deployOptions.UseHostNetwork),
					HostNetwork: deployOptions.UseHostNetwork,
					Containers: []corev1.Container{
						{
							Image:           fmt.Sprintf("%s/minio:%s", kotsadmRegistry(), kotsadmTag()),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "kotsadm-minio",
							Command: []string{
								"/bin/sh",
								"-ce",
								"/usr/bin/docker-entrypoint.sh minio -C /home/minio/.minio/ server /export",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "service",
									ContainerPort: hostnetwork.ContainerPorts(deployOptions.UseHostNetwork).MinioMinio,
									HostPort:      hostnetwork.HostPorts(deployOptions.UseHostNetwork).MinioMinio,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kotsadm-minio",
									MountPath: "/export",
								},
								{
									Name:      "minio-config-dir",
									MountPath: "/home/minio/.minio/",
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
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								FailureThreshold:    3,
								SuccessThreshold:    1,
								PeriodSeconds:       30,
								Handler: corev1.Handler{
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
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/minio/health/ready",
										Port:   intstr.FromString("service"),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
						},
					},
					InitContainers: initContainers,
				},
			},
		},
	}

	return statefulset
}

func minioService(namespace string) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: namespace,
			Labels: map[string]string{
				types.KotsadmKey: types.KotsadmLabelValue,
			},
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

// this is a pretty egregious hack to enable development against
// clusters without storage classes ready. This is primarily for
// delivering/managing CNI impls with kotsadm, since any non-cloud-specific
// volume provisioner (e.g. rook/ceph, gluster) will probably require
// a functioning pod network.
//
// Repeat: using a throwaway PV is really dangerous, but it enables
// us to get kots up and running on a single node without a pod network.
//
// somebody please make this better :)
func minioHostpathVolume() *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kotsadm-minio-host-pv",
			Labels: map[string]string{
				types.KotsadmKey: types.KotsadmLabelValue,
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: minioPVSize,
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/opt/kotsadm/minio-pv",
				},
			},
		},
	}

}
