package kotsadm

import (
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func DistributionConfigMap(deployOptions types.DeployOptions) *corev1.ConfigMap {
	labels := types.GetKotsadmLabels()
	labels["kotsadm"] = "application"

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-storage-registry-config",
			Namespace: deployOptions.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"config.yml": string(`
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
http:
  addr: :5000
  headers:
    X-Content-Type-Options:
      - nosniff
log:
  fields:
    service: registry
storage:
  cache:
    blobdescriptor: inmemory
version: 0.1`),
		},
	}

	return configMap
}

func DistributionService(deployOptions types.DeployOptions) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-storage-registry",
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm-storage-registry",
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "registry",
					Port:       5000,
					TargetPort: intstr.FromInt(5000),
				},
			},
		},
	}

	return service
}

func DistributionStatefulset(deployOptions types.DeployOptions, size resource.Quantity) *appsv1.StatefulSet {
	var securityContext *corev1.PodSecurityContext
	if !deployOptions.IsOpenShift {
		securityContext = k8sutil.SecurePodContext(1000, 1000, deployOptions.StrictSecurityContext)
	}

	podLabels := map[string]string{
		"app": "kotsadm-storage-registry",
	}
	for k, v := range deployOptions.AdditionalLabels {
		podLabels[k] = v
	}

	statefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "kotsadm-storage-registry",
			Namespace:   deployOptions.Namespace,
			Annotations: deployOptions.AdditionalAnnotations,
			Labels:      types.GetKotsadmLabels(deployOptions.AdditionalLabels),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-storage-registry",
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "kotsadm-storage-registry",
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
			ServiceName: "kotsadm-storage-registry",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: deployOptions.AdditionalAnnotations,
					Labels:      types.GetKotsadmLabels(podLabels),
				},
				Spec: corev1.PodSpec{
					SecurityContext: securityContext,
					Tolerations:     deployOptions.Tolerations,
					Containers: []corev1.Container{
						{
							Name:            "docker-registry",
							Image:           "registry:2.7.1",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"/bin/registry",
								"/serve",
								"/etc/docker/registry/config.yml",
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5000,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(5000),
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(5000),
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "REGISTRY_HTTP_SECERET",
									Value: "to-generate-",
								},
								{
									Name:  "REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY",
									Value: "/var/lib/registry",
								},
							},
							SecurityContext: k8sutil.SecureContainerContext(deployOptions.StrictSecurityContext),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kotsadm-storage-registry",
									MountPath: "/var/lib/registry",
								},
								{
									Name:      "kotsadm-storage-registry-config",
									MountPath: "/etc/docker/registry",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "kotsadm-storage-registry",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "kotsadm-storage-registry",
								},
							},
						},
						{
							Name: "kotsadm-storage-registry-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "kotsadm-storage-registry-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return statefulset
}
