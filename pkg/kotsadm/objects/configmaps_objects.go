package kotsadm

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func KotsadmConfigMap(deployOptions types.DeployOptions) *corev1.ConfigMap {
	data := map[string]string{
		"initial-app-images-pushed": fmt.Sprintf("%v", deployOptions.AppImagesPushed),
		"skip-preflights":           fmt.Sprintf("%v", deployOptions.SkipPreflights),
		"registry-is-read-only":     fmt.Sprintf("%v", deployOptions.DisableImagePush),
		"enable-image-deletion":     fmt.Sprintf("%v", deployOptions.EnableImageDeletion),
	}
	if kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.KotsadmOptions) != nil {
		data["kotsadm-registry"] = kotsadmversion.KotsadmRegistry(deployOptions.KotsadmOptions)
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      types.KotsadmConfigMap,
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Data: data,
	}

	return configMap
}

func PostgresConfigMap(deployOptions types.DeployOptions) *corev1.ConfigMap {
	// Old stretch based image used uid 999, but new alpine based image uses uid 70.
	// UID remapping is needed to allow alpine image access files created by older versions.
	data := map[string]string{
		"passwd": `root:x:0:0:root:/root:/bin/ash
postgres:x:999:999:Linux User,,,:/var/lib/postgresql:/bin/sh`,
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-postgres",
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Data: data,
	}

	return configMap
}
