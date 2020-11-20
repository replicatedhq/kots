package ingress

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/ingress/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	ConfigMapName = "kotsadm-ingress-config"
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

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, ConfigMapName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return config, nil
		}
		return nil, errors.Wrap(err, "failed to get config map")
	}

	err = yaml.Unmarshal([]byte(configMap.Data["config.yaml"]), &config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config")
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

	configMap, err := ingressConfigResource(config)
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

func ingressConfigResource(config types.Config) (*corev1.ConfigMap, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config")
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
			"config.yaml": string(data),
		},
	}, nil
}
