package generator

import (
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	shipv1beta1 "github.com/replicatedhq/ship-operator/pkg/apis/ship/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func (g *Generator) generateWebhookContainer(webhookSpec *shipv1beta1.WebhookActionSpec) corev1.Container {
	debug := level.Debug(log.With(g.logger, "method", "generateWebhookContainer"))

	toolsImageName, toolsImagePullPolicy := g.shipToolsImage()

	debug.Log("event", "construct container")
	return corev1.Container{
		Image:           toolsImageName,
		ImagePullPolicy: toolsImagePullPolicy,
		Name:            fmt.Sprintf("ship-webhook-%s", GenerateID(5)),
		Args: []string{
			"webhook",
			webhookSpec.URI,
			"--secret-namespace",
			g.instance.Namespace,
			"--secret-name",
			g.instance.Spec.State.ValueFrom.SecretKeyRef.Name,
			"--secret-key",
			g.instance.Spec.State.ValueFrom.SecretKeyRef.Key,
			"--json-payload",
			webhookSpec.Payload,
			"--directory-payload",
			"/out",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "shared",
				MountPath: "/out",
				ReadOnly:  true,
			},
		},
	}
}
