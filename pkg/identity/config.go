package identity

import (
	"context"
	"net/http"

	"github.com/coreos/go-oidc"
	dexoidc "github.com/dexidp/dex/connector/oidc"
	ghodssyaml "github.com/ghodss/yaml"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/ingress"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
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

type ErrorConfigValidation struct {
	Message string
}

func (e *ErrorConfigValidation) Error() string {
	return e.Message
}

func GetConfig(ctx context.Context, namespace string) (*kotsv1beta1.IdentityConfig, error) {
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
			return &kotsv1beta1.IdentityConfig{}, nil
		}
		return nil, errors.Wrap(err, "failed to get config map")
	}

	identityConfig, err := kotsutil.LoadIdentityConfigFromContents([]byte(configMap.Data["identity.yaml"]))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode identity config")
	}

	if err := evaluateDexConnectorsValue(ctx, namespace, &identityConfig.Spec.DexConnectors); err != nil {
		return nil, errors.Wrap(err, "failed to evaluate dex connectors value")
	}

	return identityConfig, err
}

func evaluateDexConnectorsValue(ctx context.Context, namespace string, dexConnectors *kotsv1beta1.DexConnectors) error {
	if len(dexConnectors.Value) > 0 {
		return nil
	}

	if dexConnectors.ValueFrom != nil && dexConnectors.ValueFrom.SecretKeyRef != nil {
		cfg, err := k8sconfig.GetConfig()
		if err != nil {
			return errors.Wrap(err, "failed to get kubernetes config")
		}

		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return errors.Wrap(err, "failed to get client set")
		}

		secretKeyRef := dexConnectors.ValueFrom.SecretKeyRef
		secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretKeyRef.Name, metav1.GetOptions{})
		if err != nil {
			if kuberneteserrors.IsNotFound(err) {
				return nil
			}
			return errors.Wrap(err, "failed to get secret")
		}

		err = ghodssyaml.Unmarshal(secret.Data[secretKeyRef.Key], &dexConnectors.Value)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal dex connectors")
		}
	}

	return nil
}

func SetConfig(ctx context.Context, namespace string, identityConfig kotsv1beta1.IdentityConfig) error {
	cfg, err := k8sconfig.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to get client set")
	}

	err = ensureConfigSecret(ctx, clientset, namespace, identityConfig)
	if err != nil {
		return errors.Wrap(err, "failed to ensure secret")
	}

	err = ensureConfigConfigMap(ctx, clientset, namespace, identityConfig)
	if err != nil {
		return errors.Wrap(err, "failed to ensure config map")
	}

	return nil
}

func ensureConfigConfigMap(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig) error {
	configMap, err := identityConfigMapResource(identityConfig)
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

func identityConfigMapResource(identityConfig kotsv1beta1.IdentityConfig) (*corev1.ConfigMap, error) {
	identityConfig.Spec.DexConnectors.Value = nil
	identityConfig.Spec.DexConnectors.ValueFrom = &kotsv1beta1.DexConnectorsSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: ConfigSecretName,
			},
			Key: ConfigSecretKeyName,
		},
	}

	data, err := kotsutil.EncodeIdentityConfig(identityConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode identity config")
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

func ensureConfigSecret(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig) error {
	secret, err := identitySecretResource(identityConfig)
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

func identitySecretResource(identityConfig kotsv1beta1.IdentityConfig) (*corev1.Secret, error) {
	data, err := ghodssyaml.Marshal(identityConfig.Spec.DexConnectors.Value)
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

func ConfigValidate(ctx context.Context, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig) error {
	if identityConfig.Spec.AdminConsoleAddress == "" && (!ingressConfig.Spec.Enabled || ingressConfig.Spec.Ingress == nil) {
		return &ErrorConfigValidation{Message: "adminConsoleAddress required or KOTS Admin Console ingress must be enabled"}
	}

	if identityConfig.Spec.IdentityServiceAddress == "" && (!identityConfig.Spec.IngressConfig.Enabled || identityConfig.Spec.IngressConfig.Ingress == nil) {
		return &ErrorConfigValidation{Message: "identityServiceAddress required or ingressConfig.ingress must be enabled"}
	}

	// validate kotsadm address
	if identityConfig.Spec.AdminConsoleAddress != "" {
		err := pingURL(identityConfig.Spec.AdminConsoleAddress)
		if err != nil {
			err = errors.Wrap(err, "failed to ping admin console address")
			return &ErrorConfigValidation{Message: err.Error()}
		}
	} else if ingressConfig.Spec.Enabled {
		err := pingURL(ingress.GetAddress(ingressConfig.Spec))
		if err != nil {
			err = errors.Wrap(err, "failed to ping admin console ingress")
			return &ErrorConfigValidation{Message: err.Error()}
		}
	}

	// validate dex issuer
	dexIssuer := DexIssuerURL(identityConfig.Spec)
	httpClient, err := HTTPClient(ctx, namespace, identityConfig)
	if err != nil {
		return errors.Wrap(err, "failed to init http client")
	}
	dexClientCtx := oidc.ClientContext(ctx, httpClient)
	_, err = oidc.NewProvider(dexClientCtx, dexIssuer)
	if err != nil {
		err = errors.Wrapf(err, "failed to query dex provider %q", dexIssuer)
		return &ErrorConfigValidation{Message: err.Error()}
	}

	// validate connectors issuers
	if err := evaluateDexConnectorsValue(ctx, namespace, &identityConfig.Spec.DexConnectors); err != nil {
		return errors.Wrap(err, "failed to evaluate dex connectors value")
	}
	conns, err := IdentityDexConnectorsToDexTypeConnectors(identityConfig.Spec.DexConnectors.Value)
	if err != nil {
		return errors.Wrap(err, "failed to map identity dex connectors to dex type connectors")
	}
	for _, conn := range conns {
		switch c := conn.Config.(type) {
		case *dexoidc.Config:
			_, err = oidc.NewProvider(ctx, c.Issuer)
			if err != nil {
				err = errors.Wrapf(err, "failed to query provider %q", c.Issuer)
				return &ErrorConfigValidation{Message: err.Error()}
			}
		}
	}

	return nil
}

func pingURL(url string) error {
	_, err := http.Get(url)
	if err != nil {
		return err
	}
	return nil
}
