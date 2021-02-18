package kotsadm

import (
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ApplicationMetadataConfig(data []byte, namespace string) *corev1.ConfigMap {
	labels := types.GetKotsadmLabels()
	labels["kotsadm"] = "application"

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-application-metadata",
			Namespace: namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"application.yaml": string(data),
		},
	}

	return configMap
}
