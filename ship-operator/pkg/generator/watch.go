package generator

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func (g *Generator) generateWatchContainer() corev1.Container {
	debug := level.Debug(log.With(g.logger, "method", "generateWatchContainer"))

	imageName, imagePullPolicy := g.shipImage()

	interval := g.instance.Spec.WatchInterval
	if interval == "" {
		interval = "15m"
	}

	limits := corev1.ResourceList{}
	limits[corev1.ResourceCPU] = resource.MustParse("500m")
	limits[corev1.ResourceMemory] = resource.MustParse("200Mi")

	requests := corev1.ResourceList{}
	requests[corev1.ResourceCPU] = resource.MustParse("5m")
	requests[corev1.ResourceMemory] = resource.MustParse("25Mi")

	debug.Log("event", "construct container")
	container := corev1.Container{
		Image:           imageName,
		ImagePullPolicy: imagePullPolicy,
		Name:            "ship-watch",
		Resources: corev1.ResourceRequirements{
			Limits:   limits,
			Requests: requests,
		},
		Args: []string{
			"--prefer-git",
			"--interval",
			interval,
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
			"watch",
		},
		Env: g.instance.Spec.Environment,
	}

	return container
}
