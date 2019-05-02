package generator

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func (g *Generator) generateUpdateContainer() corev1.Container {
	debug := level.Debug(log.With(g.logger, "method", "generateUpdateContainer"))

	shipImageName, shipImagePullPolicy := g.shipImage()

	limits := corev1.ResourceList{}
	limits[corev1.ResourceCPU] = resource.MustParse("500m")
	limits[corev1.ResourceMemory] = resource.MustParse("500Mi")

	requests := corev1.ResourceList{}
	requests[corev1.ResourceCPU] = resource.MustParse("100m")
	requests[corev1.ResourceMemory] = resource.MustParse("100Mi")

	debug.Log("event", "construct container")
	container := corev1.Container{
		Image:           shipImageName,
		ImagePullPolicy: shipImagePullPolicy,
		Name:            "ship-update",
		Resources: corev1.ResourceRequirements{
			Limits:   limits,
			Requests: requests,
		},
		Args: []string{
			"--prefer-git",
			"--state-from",
			"secret",
			"--secret-namespace",
			g.instance.Namespace,
			"--secret-name",
			g.instance.Spec.State.ValueFrom.SecretKeyRef.Name,
			"--secret-key",
			g.instance.Spec.State.ValueFrom.SecretKeyRef.Key,
			"--log-level",
			"debug",
			"update",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "shared",
				MountPath: "/out",
				ReadOnly:  false,
			},
		},
		Env: g.instance.Spec.Environment,
	}

	return container
}
