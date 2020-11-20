package identity

import (
	"context"

	ghodssyaml "github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/identity/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	ConfigConfigMapName = "kotsadm-identity-config"
	ConfigSecretName    = "kotsadm-identity-secret"
)

func GetConfig(ctx context.Context, namespace string) (*types.Config, error) {
	cfg, err := k8sconfig.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client set")
	}

	config := &types.Config{}

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, ConfigConfigMapName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return config, nil
		}
		return nil, errors.Wrap(err, "failed to get config map")
	}

	err = ghodssyaml.Unmarshal([]byte(configMap.Data["config.yaml"]), config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config")
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, ConfigSecretName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return config, nil
		}
		return nil, errors.Wrap(err, "failed to get secret")
	}

	err = ghodssyaml.Unmarshal(secret.Data["dexConnectors"], &config.DexConnectors)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal dex connectors")
	}

	return config, err
}

func SetConfig(ctx context.Context, namespace string, config types.Config) error {
	cfg, err := k8sconfig.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to get client set")
	}

	err = ensureConfigSecret(ctx, clientset, namespace, config)
	if err != nil {
		return errors.Wrap(err, "failed to ensure secret")
	}

	err = ensureConfigConfigMap(ctx, clientset, namespace, config)
	if err != nil {
		return errors.Wrap(err, "failed to ensure config map")
	}

	return nil
}

func ensureConfigConfigMap(ctx context.Context, clientset kubernetes.Interface, namespace string, config types.Config) error {
	configMap, err := identityConfigMapResource(config)
	if err != nil {
		return err
	}

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, ConfigConfigMapName, metav1.GetOptions{})
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

func identityConfigMapResource(config types.Config) (*corev1.ConfigMap, error) {
	config.DexConnectors = nil // stored in a secret

	data, err := ghodssyaml.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config")
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ConfigConfigMapName,
			Labels: kotsadmtypes.GetKotsadmLabels(map[string]string{
				KotsIdentityLabelKey: KotsIdentityLabelValue,
			}),
		},
		Data: map[string]string{
			"config.yaml": string(data),
		},
	}, nil
}

func ensureConfigSecret(ctx context.Context, clientset kubernetes.Interface, namespace string, config types.Config) error {
	secret, err := identitySecretResource(config)
	if err != nil {
		return err
	}

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, ConfigSecretName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get secret")
		}

		_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}

		return nil
	}

	existingSecret.Data = secret.Data

	_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update secret")
	}

	return nil
}

func identitySecretResource(config types.Config) (*corev1.Secret, error) {
	data, err := ghodssyaml.Marshal(config.DexConnectors)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal secret")
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ConfigSecretName,
			Labels: kotsadmtypes.GetKotsadmLabels(map[string]string{
				KotsIdentityLabelKey: KotsIdentityLabelValue,
			}),
		},
		Data: map[string][]byte{
			"dexConnectors": data,
		},
	}, nil
}
