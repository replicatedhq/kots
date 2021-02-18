package kotsadm

import (
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
)

func GetProxyEnv(deployOptions types.DeployOptions) []corev1.EnvVar {
	result := []corev1.EnvVar{
		{
			Name:  "HTTP_PROXY",
			Value: deployOptions.HTTPProxyEnvValue,
		},
		{
			Name:  "HTTPS_PROXY",
			Value: deployOptions.HTTPSProxyEnvValue,
		},
	}

	kotsadmNoProxy := "kotsadm-postgres,kotsadm-minio,kotsadm-api-node"
	if deployOptions.NoProxyEnvValue == "" {
		result = append(result, corev1.EnvVar{
			Name:  "NO_PROXY",
			Value: kotsadmNoProxy,
		})
	} else {
		result = append(result, corev1.EnvVar{
			Name:  "NO_PROXY",
			Value: deployOptions.NoProxyEnvValue + "," + kotsadmNoProxy,
		})
	}

	return result
}
