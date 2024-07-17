package kotsadm

import (
	_ "embed"
	"fmt"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func KotsadmConfigMap(deployOptions types.DeployOptions) *corev1.ConfigMap {
	data := map[string]string{
		"kots-install-id":           fmt.Sprintf("%v", deployOptions.InstallID),
		"initial-app-images-pushed": fmt.Sprintf("%v", deployOptions.AppImagesPushed),
		"skip-preflights":           fmt.Sprintf("%v", deployOptions.SkipPreflights),
		"registry-is-read-only":     fmt.Sprintf("%v", deployOptions.DisableImagePush),
		"minio-enabled-snapshots":   fmt.Sprintf("%v", deployOptions.IncludeMinioSnapshots),
		"skip-compatibility-check":  fmt.Sprintf("%v", deployOptions.SkipCompatibilityCheck),
		"ensure-rbac":               fmt.Sprintf("%v", deployOptions.EnsureRBAC),
		"skip-rbac-check":           fmt.Sprintf("%v", deployOptions.SkipRBACCheck),
		"use-minimal-rbac":          fmt.Sprintf("%v", deployOptions.UseMinimalRBAC),
		"strict-security-context":   fmt.Sprintf("%v", deployOptions.StrictSecurityContext),
		"wait-duration":             fmt.Sprintf("%v", deployOptions.Timeout),
		"with-minio":                fmt.Sprintf("%v", deployOptions.IncludeMinio),
		"app-version-label":         deployOptions.AppVersionLabel,
		"requested-channel-id":      deployOptions.RequestedChannelID,
	}

	if kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.RegistryConfig) != nil {
		data["kotsadm-registry"] = kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig)
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
