package identity

import (
	"context"
	"net/http"

	"crypto/tls"

	"github.com/coreos/go-oidc"
	dexoidc "github.com/dexidp/dex/connector/oidc"
	ghodssyaml "github.com/ghodss/yaml"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	ConfigConfigMapName = "kotsadm-identity-config"
	ConfigSecretName    = "kotsadm-identity-secret"
	ConfigSecretKeyName = "dexConnectors"
)

var insecureClient *http.Client

func init() {
	transport := cleanhttp.DefaultTransport()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	insecureClient = &http.Client{
		Transport: transport,
	}
}

type ErrorConnection struct {
	Message string
}

func (e *ErrorConnection) Error() string {
	return e.Message
}

func GetConfig(ctx context.Context, namespace string) (*kotsv1beta1.IdentityConfig, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s client set")
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

	return identityConfig, nil
}

func evaluateDexConnectorsValue(ctx context.Context, namespace string, dexConnectors *kotsv1beta1.DexConnectors) error {
	if len(dexConnectors.Value) > 0 {
		return nil
	}

	if dexConnectors.ValueFrom != nil && dexConnectors.ValueFrom.SecretKeyRef != nil {
		clientset, err := k8sutil.GetClientset()
		if err != nil {
			return errors.Wrap(err, "failed to get k8s client set")
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
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
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
	// NOTE: we do not encrypt kotsadm config

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
			Name:   ConfigConfigMapName,
			Labels: kotsadmtypes.GetKotsadmLabels(identitydeploy.AdditionalLabels("kotsadm", nil)),
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
	// NOTE: we do not encrypt kotsadm config

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
			Name:   ConfigSecretName,
			Labels: kotsadmtypes.GetKotsadmLabels(identitydeploy.AdditionalLabels("kotsadm", nil)),
		},
		Data: map[string][]byte{
			ConfigSecretKeyName: data,
		},
	}, nil
}

func ValidateConfig(ctx context.Context, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig) error {
	if identityConfig.Spec.AdminConsoleAddress == "" && (!ingressConfig.Spec.Enabled || ingressConfig.Spec.Ingress == nil) {
		return errors.New("adminConsoleAddress required or KOTS Admin Console ingress must be enabled")
	}

	if identityConfig.Spec.IdentityServiceAddress == "" && (!identityConfig.Spec.IngressConfig.Enabled || identityConfig.Spec.IngressConfig.Ingress == nil) {
		return errors.New("identityServiceAddress required or ingressConfig.ingress must be enabled")
	}

	return nil
}

func ValidateConnection(ctx context.Context, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig) error {
	// validate kotsadm address
	if identityConfig.Spec.AdminConsoleAddress != "" {
		err := pingURL(identityConfig.Spec.AdminConsoleAddress)
		if err != nil {
			err = errors.Wrap(err, "failed to ping admin console address")
			return &ErrorConnection{Message: err.Error()}
		}
	} else if ingressConfig.Spec.Enabled {
		err := pingURL(ingress.GetAddress(ingressConfig.Spec))
		if err != nil {
			err = errors.Wrap(err, "failed to ping admin console ingress")
			return &ErrorConnection{Message: err.Error()}
		}
	}

	// TODO: make this work, the challenge is waiting for the dex pods to become ready/be deployed before validating
	// validate dex issuer
	/**
		dexIssuer := DexIssuerURL(identityConfig.Spec)
		httpClient, err := HTTPClient(ctx, namespace, identityConfig)
		if err != nil {
			return errors.Wrap(err, "failed to init http client")
		}
		dexClientCtx := oidc.ClientContext(ctx, httpClient)
		_, err = oidc.NewProvider(dexClientCtx, dexIssuer)
		if err != nil {
			err = errors.Wrapf(err, "failed to query dex provider %q", dexIssuer)
			return &ErrorConnection{Message: err.Error()}
		}
	**/

	// NOTE: we do not encrypt kotsadm config

	// validate connectors issuers
	if err := evaluateDexConnectorsValue(ctx, namespace, &identityConfig.Spec.DexConnectors); err != nil {
		return errors.Wrap(err, "failed to evaluate dex connectors value")
	}
	conns, err := identitydeploy.DexConnectorsToDexTypeConnectors(identityConfig.Spec.DexConnectors.Value)
	if err != nil {
		return errors.Wrap(err, "failed to map identity dex connectors to dex type connectors")
	}
	for _, conn := range conns {
		switch c := conn.Config.(type) {
		case *dexoidc.Config:
			_, err = oidc.NewProvider(ctx, c.Issuer)
			if err != nil {
				err = errors.Wrapf(err, "failed to query provider %q", c.Issuer)
				return &ErrorConnection{Message: err.Error()}
			}
		}
	}

	return nil
}

func pingURL(url string) error {
	_, err := insecureClient.Get(url)
	if err != nil {
		return err
	}
	return nil
}
