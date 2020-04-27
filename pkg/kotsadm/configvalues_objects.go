package kotsadm

import (
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func configValuesSecret(namespace string, configValues string) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-default-configvalues",
			Namespace: namespace,
			Labels: map[string]string{
				types.KotsadmKey:     types.KotsadmLabelValue,
				"kots.io/automation": "configvalues",
				types.VeleroKey:      types.VeleroLabelValue,
			},
		},
		Data: map[string][]byte{
			"configvalues": []byte(configValues),
		},
	}

	return secret
}
