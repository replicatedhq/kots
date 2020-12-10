package midstream

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/dexidp/dex/server"
	dexstorage "github.com/dexidp/dex/storage"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/segmentio/ksuid"
	yaml "gopkg.in/yaml.v2"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func (m *Midstream) writeIdentityService(ctx context.Context, options WriteOptions) (string, error) {
	if m.IdentityConfig == nil {
		return "", nil
	}

	dexConfig, err := getDexConfig(ctx, m.IdentityConfig.Spec)
	if err != nil {
		return "", errors.Wrap(err, "failed to get dex config")
	}

	resources, err := identitydeploy.Render(ctx, options.AppSlug, dexConfig, m.IdentityConfig.Spec.IngressConfig, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to render identity service")
	}

	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		CommonLabels: map[string]string{
			// TODO (ethan)
		},
	}

	base := "identity-service"

	absDir := filepath.Join(options.MidstreamDir, base)
	if err := os.MkdirAll(absDir, 0744); err != nil {
		return "", errors.Wrap(err, "failed to mkdir")
	}

	for filename, resource := range resources {
		if err := ioutil.WriteFile(filepath.Join(absDir, filename), resource, 0644); err != nil {
			return "", errors.Wrapf(err, "failed to write resource %s", filename)
		}
		kustomization.Resources = append(kustomization.Resources, filename)
	}

	if err := k8sutil.WriteKustomizationToFile(kustomization, filepath.Join(absDir, "kustomization.yaml")); err != nil {
		return "", errors.Wrap(err, "failed to write kustomization file")
	}

	return base, nil
}

func getDexConfig(ctx context.Context, identityConfigSpec kotsv1beta1.IdentityConfigSpec) ([]byte, error) {
	postgresPassword, err := getDexPostgresPassword(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dex postgres password")
	}

	staticClient, err := getOIDCClient(ctx, identityConfigSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get oidc client")
	}

	config := dextypes.Config{
		Issuer: dexIssuerURL(identityConfigSpec),
		Storage: dextypes.Storage{
			Type: "postgres",
			Config: dextypes.Postgres{
				NetworkDB: dextypes.NetworkDB{
					Database: "dex",
					User:     "dex",
					Host:     "kotsadm-postgres",
					Password: postgresPassword,
				},
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

	if len(identityConfigSpec.DexConnectors.Value) > 0 {
		dexConnectors, err := identitydeploy.DexConnectorsToDexTypeConnectors(identityConfigSpec.DexConnectors.Value)
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
		"OIDCIdentityCallbackURL": func() string { return dexCallbackURL(identityConfigSpec) },
	}).Parse(string(marshalledConfig))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse dex config for templating")
	}
	if err := t.Execute(buf, nil); err != nil {
		return nil, errors.Wrap(err, "failed to execute template")
	}

	return buf.Bytes(), nil
}

func dexIssuerURL(identityConfigSpec kotsv1beta1.IdentityConfigSpec) string {
	if identityConfigSpec.IdentityServiceAddress != "" {
		return identityConfigSpec.IdentityServiceAddress
	}
	return fmt.Sprintf("%s/dex", ingress.GetAddress(identityConfigSpec.IngressConfig))
}

func dexCallbackURL(identityConfigSpec kotsv1beta1.IdentityConfigSpec) string {
	return fmt.Sprintf("%s/callback", dexIssuerURL(identityConfigSpec))
}

func getOIDCClient(ctx context.Context, identityConfigSpec kotsv1beta1.IdentityConfigSpec) (dexstorage.Client, error) {
	clientSecret := ksuid.New().String()

	// TODO (ethan): find existing secret from idk where
	// do not assume we have access to the clustr

	return dexstorage.Client{
		ID:           "kotsadm",
		Name:         "kotsadm",
		Secret:       clientSecret,
		RedirectURIs: []string{
			// TODO: pass identity spec through to here
		},
	}, nil
}

func getDexPostgresPassword(ctx context.Context) (string, error) {
	// TODO (ethan): this probably has to be passed in as a secret or something
	// do not assume we have access to the clustr

	return "TODO", nil
}
