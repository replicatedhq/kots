package ingress

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ConfigMapName = "kotsadm-ingress-config"
)

func GetConfig(ctx context.Context, namespace string) (*kotsv1beta1.IngressConfig, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s client set")
	}

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, ConfigMapName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return &kotsv1beta1.IngressConfig{}, nil
		}
		return nil, errors.Wrap(err, "failed to get config map")
	}

	ingressConfig, err := kotsutil.LoadIngressConfigFromContents([]byte(configMap.Data["ingress.yaml"]))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode ingress config")
	}

	return ingressConfig, err
}

func SetConfig(ctx context.Context, namespace string, ingressConfig kotsv1beta1.IngressConfig) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	configMap, err := ingressConfigResource(ingressConfig)
	if err != nil {
		return err
	}

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, ConfigMapName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get config map")
		}

		_, err = clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create config map")
		}

		return nil
	}

	existingConfigMap.Data = configMap.Data

	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

func ingressConfigResource(ingressConfig kotsv1beta1.IngressConfig) (*corev1.ConfigMap, error) {
	data, err := kotsutil.EncodeIngressConfig(ingressConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode ingress config")
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   ConfigMapName,
			Labels: kotsadmtypes.GetKotsadmLabels(),
		},
		Data: map[string]string{
			"ingress.yaml": string(data),
		},
	}, nil
}
