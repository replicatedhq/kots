package kotsadm

import (
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ConfigValuesSecret(namespace string, configValues string) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-default-configvalues",
			Namespace: namespace,
			Labels: types.GetKotsadmLabels(map[string]string{
				"kots.io/automation": "configvalues",
			}),
		},
		Data: map[string][]byte{
			"configvalues": []byte(configValues),
		},
	}

	return secret
}
