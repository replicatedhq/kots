package supportbundle

import (
	"context"
	"regexp"

	"github.com/pkg/errors"
	kotstypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var legacySupportBundleSpecConfigMapNameRE = regexp.MustCompile(`^kotsadm-.*-supportbundle-(vendor|cluster-specific|default)$`)
var legacyRedactorSpecConfigMapNameRE = regexp.MustCompile(`^kotsadm(-.*)?-redact(-.*)?-spec$`)

// CleanupLegacySpecConfigMaps deletes the legacy ConfigMaps that were used to store
// support bundle sub-specs before they were moved to Secrets. It is idempotent and
// safe to run on every admin-console startup.
func CleanupLegacySpecConfigMaps(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	labelSelector := kotstypes.TroubleshootKey + "=" + kotstypes.TroubleshootValue

	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return errors.Wrap(err, "failed to list legacy support bundle spec configmaps")
	}

	for _, configmap := range configmaps.Items {
		if !legacySupportBundleSpecConfigMapNameRE.MatchString(configmap.Name) {
			continue
		}

		err := clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, configmap.Name, metav1.DeleteOptions{})
		if err != nil {
			if !kuberneteserrors.IsNotFound(err) {
				logger.Errorf("failed to delete legacy support bundle spec configmap %s: %v", configmap.Name, err)
			}
			continue
		}

		logger.Infof("deleted legacy support bundle spec configmap %s", configmap.Name)
	}

	return nil
}

// CleanupLegacyRedactSpecConfigMaps deletes the legacy ConfigMaps that were used to store
// rendered redactor specs before they were moved to Secrets. It is idempotent and
// safe to run on every admin-console startup.
func CleanupLegacyRedactSpecConfigMaps(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	labelSelector := kotstypes.KotsadmKey + "=" + kotstypes.KotsadmLabelValue

	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return errors.Wrap(err, "failed to list legacy redactor spec configmaps")
	}

	for _, configmap := range configmaps.Items {
		if !legacyRedactorSpecConfigMapNameRE.MatchString(configmap.Name) {
			continue
		}

		err := clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, configmap.Name, metav1.DeleteOptions{})
		if err != nil {
			if !kuberneteserrors.IsNotFound(err) {
				logger.Errorf("failed to delete legacy redactor spec configmap %s: %v", configmap.Name, err)
			}
			continue
		}

		logger.Infof("deleted legacy redactor spec configmap %s", configmap.Name)
	}

	return nil
}
