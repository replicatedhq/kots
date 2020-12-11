package identity

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/dexidp/dex/server"
	dexstorage "github.com/dexidp/dex/storage"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/segmentio/ksuid"
	"gopkg.in/yaml.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func getDexConfig(ctx context.Context, clientset kubernetes.Interface, namespace string, identitySpec kotsv1beta1.IdentityConfigSpec, ingressSpec kotsv1beta1.IngressConfigSpec) ([]byte, error) {
	staticClient, err := getOIDCClient(ctx, clientset, namespace, identitySpec, ingressSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get oidc client")
	}

	config := dextypes.Config{
		Issuer: DexIssuerURL(identitySpec),
		Storage: dextypes.Storage{
			Type: "postgres",
			Config: dextypes.Postgres{
				SSL: dextypes.SSL{
					Mode: "disable", // TODO ssl
				},
			},
		},
		Web: dextypes.Web{
			HTTP: "0.0.0.0:5556",
		},
		Frontend: server.WebConfig{
			Issuer: "KOTS",
		},
		OAuth2: dextypes.OAuth2{
			SkipApprovalScreen:    true,
			AlwaysShowLoginScreen: false, // possibly make this configurable
		},
		StaticClients:    []dexstorage.Client{staticClient},
		EnablePasswordDB: false,
	}

	if len(identitySpec.DexConnectors.Value) > 0 {
		dexConnectors, err := identitydeploy.DexConnectorsToDexTypeConnectors(identitySpec.DexConnectors.Value)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal dex connectors")
		}
		config.StaticConnectors = dexConnectors
	}

	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate dex config")
	}

	marshalledConfig, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal dex config")
	}

	buf := bytes.NewBuffer(nil)
	t, err := template.New("dex-config").Funcs(template.FuncMap{
		"OIDCIdentityCallbackURL": func() string { return DexCallbackURL(identitySpec) },
	}).Parse(string(marshalledConfig))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse dex config for templating")
	}
	if err := t.Execute(buf, nil); err != nil {
		return nil, errors.Wrap(err, "failed to execute template")
	}

	return buf.Bytes(), nil
}

func getOIDCClient(ctx context.Context, clientset kubernetes.Interface, namespace string, identitySpec kotsv1beta1.IdentityConfigSpec, ingressSpec kotsv1beta1.IngressConfigSpec) (dexstorage.Client, error) {
	existingConfig, err := getKotsadmDexConfig(ctx, clientset, namespace)
	if err != nil {
		return dexstorage.Client{}, errors.Wrap(err, "failed to get existing dex config")
	}

	kotsadmAddress := identitySpec.AdminConsoleAddress
	if kotsadmAddress == "" && ingressSpec.Enabled {
		kotsadmAddress = ingress.GetAddress(ingressSpec)
	}

	clientSecret := ksuid.New().String()
	for _, client := range existingConfig.StaticClients {
		if client.ID == "kotsadm" {
			clientSecret = client.Secret
			break
		}
	}

	return dexstorage.Client{
		ID:     "kotsadm",
		Name:   "kotsadm",
		Secret: clientSecret,
		RedirectURIs: []string{
			fmt.Sprintf("%s/api/v1/oidc/login/callback", kotsadmAddress),
		},
	}, nil
}

func getKotsadmDexConfig(ctx context.Context, clientset kubernetes.Interface, namespace string) (*dextypes.Config, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, "kotsadm-dex", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm-dex secret")
	}

	marshalledConfig := secret.Data["dexConfig.yaml"]
	config := dextypes.Config{}
	if err := yaml.Unmarshal(marshalledConfig, &config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal kotsadm dex config")
	}

	return &config, nil
}
