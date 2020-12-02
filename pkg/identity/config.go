package identity

import (
	"context"

	ghodssyaml "github.com/ghodss/yaml"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
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
	ConfigSecretKeyName = "dexConnectors"
)

func GetConfig(ctx context.Context, namespace string) (*kotsv1beta1.Identity, error) {
	cfg, err := k8sconfig.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client set")
	}

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, ConfigConfigMapName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return &kotsv1beta1.Identity{}, nil
		}
		return nil, errors.Wrap(err, "failed to get config map")
	}

	spec, err := DecodeSpec([]byte(configMap.Data["identity.yaml"]))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode spec")
	}

	if spec.Spec.DexConnectors.ValueFrom != nil && spec.Spec.DexConnectors.ValueFrom.SecretKeyRef != nil {
		secretKeyRef := spec.Spec.DexConnectors.ValueFrom.SecretKeyRef

		secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretKeyRef.Name, metav1.GetOptions{})
		if err != nil {
			if kuberneteserrors.IsNotFound(err) {
				return spec, nil
			}
			return nil, errors.Wrap(err, "failed to get secret")
		}

		err = ghodssyaml.Unmarshal(secret.Data[secretKeyRef.Key], &spec.Spec.DexConnectors.Value)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal dex connectors")
		}
	}

	return spec, err
}

func SetConfig(ctx context.Context, namespace string, spec kotsv1beta1.Identity) error {
	cfg, err := k8sconfig.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to get client set")
	}

	err = ensureConfigSecret(ctx, clientset, namespace, spec)
	if err != nil {
		return errors.Wrap(err, "failed to ensure secret")
	}

	err = ensureConfigConfigMap(ctx, clientset, namespace, spec)
	if err != nil {
		return errors.Wrap(err, "failed to ensure config map")
	}

	return nil
}

func ensureConfigConfigMap(ctx context.Context, clientset kubernetes.Interface, namespace string, spec kotsv1beta1.Identity) error {
	configMap, err := identityConfigMapResource(spec)
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

func identityConfigMapResource(spec kotsv1beta1.Identity) (*corev1.ConfigMap, error) {
	spec.Spec.DexConnectors.Value = nil
	spec.Spec.DexConnectors.ValueFrom = &kotsv1beta1.DexConnectorsSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: ConfigSecretName,
			},
			Key: ConfigSecretKeyName,
		},
	}

	data, err := EncodeSpec(spec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode spec")
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
			"identity.yaml": string(data),
		},
	}, nil
}

func ensureConfigSecret(ctx context.Context, clientset kubernetes.Interface, namespace string, spec kotsv1beta1.Identity) error {
	secret, err := identitySecretResource(spec)
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

func identitySecretResource(spec kotsv1beta1.Identity) (*corev1.Secret, error) {
	data, err := ghodssyaml.Marshal(spec.Spec.DexConnectors.Value)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal dex connectors")
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
			ConfigSecretKeyName: data,
		},
	}, nil
}
