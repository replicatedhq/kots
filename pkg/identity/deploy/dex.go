package deploy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dexidp/dex/connector/oidc"
	"github.com/dexidp/dex/server"
	dexstorage "github.com/dexidp/dex/storage"
	ghodssyaml "github.com/ghodss/yaml"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
	"github.com/replicatedhq/kots/pkg/template"
)

func getDexConfig(ctx context.Context, issuerURL string, options Options) ([]byte, error) {
	identitySpec := options.IdentitySpec
	identityConfigSpec := options.IdentityConfigSpec
	builder := options.Builder
	cipher := options.Cipher

	redirectURIs, err := buildIdentitySpecOIDCRedirectURIs(identitySpec.OIDCRedirectURIs, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build identity spec oicd redirect uris")
	}

	config := dextypes.Config{
		Issuer: issuerURL,
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
			AlwaysShowLoginScreen: identitySpec.OAUTH2AlwaysShowLoginScreen,
		},
		Expiry: dextypes.Expiry{
			IDTokens:    identitySpec.IDTokensExpiration,
			SigningKeys: identitySpec.SigningKeysExpiration,
		},
		StaticClients: []dexstorage.Client{
			{
				ID:           identityConfigSpec.ClientID,
				Name:         identityConfigSpec.ClientID,
				SecretEnv:    "DEX_CLIENT_SECRET",
				RedirectURIs: redirectURIs,
			},
		},
		EnablePasswordDB: false,
	}

	dexConnectors := []kotsv1beta1.DexConnector{}
	if cipher != nil {
		dexConnectors, err = identityConfigSpec.DexConnectors.GetValue(*cipher)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decrypt dex connectors")
		}
	} else if identityConfigSpec.DexConnectors.ValueEncrypted != "" {
		return nil, errors.New("cipher required")
	} else {
		// NOTE: we do not encrypt kotsadm config
		dexConnectors = identityConfigSpec.DexConnectors.Value
	}

	connectors := []kotsv1beta1.DexConnector{}
	for _, connector := range dexConnectors {
		if len(identitySpec.SupportedProviders) == 0 || stringInSlice(connector.Type, identitySpec.SupportedProviders) {
			connectors = append(connectors, connector)
		}
	}

	if len(connectors) == 0 {
		return nil, errors.New("at lease one dex connector is required")
	}

	if len(connectors) > 0 {
		dexConnectors, err := DexConnectorsToDexTypeConnectors(connectors)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal dex connectors")
		}
		config.StaticConnectors = dexConfigReplaceDynamicValues(issuerURL, dexConnectors)
	}

	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate dex config")
	}

	marshalledConfig, err := ghodssyaml.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal dex config")
	}

	return marshalledConfig, nil
}

func dexConfigReplaceDynamicValues(issuerURL string, connectors []dextypes.Connector) []dextypes.Connector {
	next := make([]dextypes.Connector, len(connectors))
	for i, connector := range connectors {
		switch c := connector.Config.(type) {
		case *oidc.Config:
			c.RedirectURI = dexCallbackURL(issuerURL)
		}
		next[i] = connector
	}
	return next
}

func DexConnectorsToDexTypeConnectors(conns []kotsv1beta1.DexConnector) ([]dextypes.Connector, error) {
	dexConnectors := []dextypes.Connector{}
	for _, conn := range conns {
		f, ok := server.ConnectorsConfig[conn.Type]
		if !ok {
			return nil, errors.Errorf("unknown connector type %q", conn.Type)
		}

		connConfig := f()
		if len(conn.Config.Raw) != 0 {
			if err := json.Unmarshal(conn.Config.Raw, connConfig); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal connector config")
			}
		}

		dexConnectors = append(dexConnectors, dextypes.Connector{
			Type:   conn.Type,
			Name:   conn.Name,
			ID:     conn.ID,
			Config: connConfig,
		})
	}
	return dexConnectors, nil
}

func dexIssuerURL(identitySpec kotsv1beta1.IdentitySpec, builder *template.Builder) (string, error) {
	// TODO: ingress
	if builder == nil {
		return identitySpec.IdentityIssuerURL, nil
	}
	return builder.String(identitySpec.IdentityIssuerURL)
}

func dexCallbackURL(issuerURL string) string {
	return fmt.Sprintf("%s/callback", issuerURL)
}

func buildIdentitySpecOIDCRedirectURIs(uris []string, builder *template.Builder) ([]string, error) {
	if builder == nil {
		return uris, nil
	}

	next := []string{}
	for _, uri := range uris {
		rendered, err := builder.String(uri)
		if err != nil {
			return nil, errors.Wrapf(err, "build %q", uri)
		}
		if rendered != "" {
			next = append(next, rendered)
		}
	}
	return next, nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
