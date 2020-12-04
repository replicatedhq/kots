package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/dexidp/dex/connector/oidc"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/identity"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	RedactionMask = "--- REDACTED ---"
)

type ConfigureIdentityServiceRequest struct {
	AdminConsoleAddress    string `json:"adminConsoleAddress"`
	IdentityServiceAddress string `json:"identityServiceAddress"`

	IDPConfig `json:",inline"`
}

type IDPConfig struct {
	OIDCConfig    *OIDCConfig `json:"oidcConfig"`
	GEOAxISConfig *OIDCConfig `json:"geoAxisConfig"`
}
type OIDCConfig struct {
	ConnectorID               string           `json:"connectorId"`
	ConnectorName             string           `json:"connectorName"`
	Issuer                    string           `json:"issuer"`
	ClientID                  string           `json:"clientId"`
	ClientSecret              string           `json:"clientSecret"`
	GetUserInfo               *bool            `json:"getUserInfo,omitempty"`
	UserNameKey               string           `json:"userNameKey,omitempty"`
	UserIDKey                 string           `json:"userIDKey,omitempty"`
	PromptType                string           `json:"promptType,omitempty"`
	InsecureSkipEmailVerified *bool            `json:"insecureSkipEmailVerified,omitempty"`
	InsecureEnableGroups      *bool            `json:"insecureEnableGroups,omitempty"`
	Scopes                    []string         `json:"scopes,omitempty"`
	HostedDomains             []string         `json:"hostedDomains,omitempty"`
	ClaimMapping              OIDCClaimMapping `json:"claimMapping,omitempty"`
}
type OIDCClaimMapping struct {
	PreferredUsernameKey string `json:"preferredUsername,omitempty"`
	EmailKey             string `json:"email,omitempty"`
	GroupsKey            string `json:"groups,omitempty"`
}

func ConfigureIdentityService(w http.ResponseWriter, r *http.Request) {
	request := ConfigureIdentityServiceRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		err = errors.Wrap(err, "failed to decode request body")
		logger.Error(err)
		JSON(w, http.StatusBadRequest, NewErrorResponse(err))
		return
	}

	if err := validateConfigureIdentityRequest(request); err != nil {
		err = errors.Wrap(err, "failed to validate request")
		logger.Error(err)
		JSON(w, http.StatusBadRequest, NewErrorResponse(err))
		return
	}

	namespace := os.Getenv("POD_NAMESPACE")

	previousConfig, err := identity.GetConfig(r.Context(), namespace)
	if err != nil {
		err = errors.Wrap(err, "failed to get identity config")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	idpConfigs, err := dexConnectorsToIDPConfigs(previousConfig.Spec.DexConnectors.Value)
	if err != nil {
		err = errors.Wrap(err, "failed to get idp configs from dex connectors")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	connectorInfo, err := getDexConnectorInfo(request, idpConfigs)
	if err != nil {
		err = errors.Wrap(err, "failed to get dex connector info")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: handle ingress config
	identityConfig := kotsv1beta1.IdentityConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "IdentityConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "identity",
		},
		Spec: kotsv1beta1.IdentityConfigSpec{
			Enabled:                true,
			DisablePasswordAuth:    true,
			AdminConsoleAddress:    request.AdminConsoleAddress,
			IdentityServiceAddress: request.IdentityServiceAddress,
			DexConnectors: kotsv1beta1.DexConnectors{
				Value: []kotsv1beta1.DexConnector{
					{
						Type: connectorInfo.Type,
						Name: connectorInfo.Name,
						ID:   connectorInfo.ID,
						Config: runtime.RawExtension{
							Raw: connectorInfo.Config,
						},
					},
				},
			},
		},
	}

	ingressConfig, err := ingress.GetConfig(r.Context(), namespace)
	if err != nil {
		err = errors.Wrap(err, "failed to get ingress config")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := identity.ConfigValidate(r.Context(), namespace, identityConfig, *ingressConfig); err != nil {
		if _, ok := errors.Cause(err).(*identity.ErrorConfigValidation); ok {
			err = errors.Wrap(err, "invalid identity config")
			logger.Error(err)
			JSON(w, http.StatusBadRequest, NewErrorResponse(err))
			return
		}
		err = errors.Wrap(err, "failed to validate identity config")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := identity.SetConfig(r.Context(), namespace, identityConfig); err != nil {
		err = errors.Wrap(err, "failed to set identity config")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	cfg, err := config.GetConfig()
	if err != nil {
		err = errors.Wrap(err, "failed to get cluster config")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		err = errors.Wrap(err, "failed to create kubernetes clientset")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	registryOptions, err := kotsadm.GetKotsadmOptionsFromCluster(namespace, clientset)
	if err != nil {
		err = errors.Wrap(err, "failed to get kotsadm options from cluster")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := identity.Deploy(r.Context(), clientset, namespace, identityConfig, *ingressConfig, &registryOptions); err != nil {
		err = errors.Wrap(err, "failed to deploy the identity service")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := ErrorResponse{
		Success: true,
	}

	JSON(w, http.StatusOK, response)
}

type DexConnectorInfo struct {
	Type   string
	ID     string
	Name   string
	Config []byte
}

func getDexConnectorInfo(request ConfigureIdentityServiceRequest, idpConfigs []IDPConfig) (*DexConnectorInfo, error) {
	var connectorConfig interface{}
	dexConnectorInfo := DexConnectorInfo{}

	if request.OIDCConfig != nil {
		c := oidc.Config{
			Issuer:                    request.OIDCConfig.Issuer,
			ClientID:                  request.OIDCConfig.ClientID,
			ClientSecret:              request.OIDCConfig.ClientSecret,
			RedirectURI:               "{{OIDCIdentityCallbackURL}}",
			GetUserInfo:               true,
			UserNameKey:               "email",
			InsecureSkipEmailVerified: false,
			InsecureEnableGroups:      true,
			Scopes: []string{
				"openid",
				"email",
				"groups",
			},
		}

		// un-redact
		if c.ClientSecret == RedactionMask {
			for _, idpConfig := range idpConfigs {
				if idpConfig.OIDCConfig != nil {
					c.ClientSecret = idpConfig.OIDCConfig.ClientSecret
				}
			}
		}

		// overrides and advanced options
		if request.OIDCConfig.GetUserInfo != nil {
			c.GetUserInfo = *request.OIDCConfig.GetUserInfo
		}
		if request.OIDCConfig.UserNameKey != "" {
			c.UserNameKey = request.OIDCConfig.UserNameKey
		}
		if request.OIDCConfig.UserIDKey != "" {
			c.UserIDKey = request.OIDCConfig.UserIDKey
		}
		if request.OIDCConfig.PromptType != "" {
			c.PromptType = request.OIDCConfig.PromptType
		}
		if request.OIDCConfig.InsecureSkipEmailVerified != nil {
			c.InsecureSkipEmailVerified = *request.OIDCConfig.InsecureSkipEmailVerified
		}
		if request.OIDCConfig.InsecureEnableGroups != nil {
			c.InsecureEnableGroups = *request.OIDCConfig.InsecureEnableGroups
		}
		if len(request.OIDCConfig.Scopes) > 0 {
			c.Scopes = append(c.Scopes, request.OIDCConfig.Scopes...)
		}
		if len(request.OIDCConfig.HostedDomains) > 0 {
			c.HostedDomains = request.OIDCConfig.HostedDomains
		}
		if request.OIDCConfig.ClaimMapping.PreferredUsernameKey != "" {
			c.ClaimMapping.PreferredUsernameKey = request.OIDCConfig.ClaimMapping.PreferredUsernameKey
		}
		if request.OIDCConfig.ClaimMapping.EmailKey != "" {
			c.ClaimMapping.EmailKey = request.OIDCConfig.ClaimMapping.EmailKey
		}
		if request.OIDCConfig.ClaimMapping.GroupsKey != "" {
			c.ClaimMapping.GroupsKey = request.OIDCConfig.ClaimMapping.GroupsKey
		}
		connectorConfig = c

		dexConnectorInfo.Type = "oidc"
		dexConnectorInfo.ID = "openid"
		dexConnectorInfo.Name = "OpenID"

		if request.OIDCConfig.ConnectorID != "" {
			dexConnectorInfo.ID = request.OIDCConfig.ConnectorID
		}
		if request.OIDCConfig.ConnectorName != "" {
			dexConnectorInfo.Name = request.OIDCConfig.ConnectorName
		}
	} else if request.GEOAxISConfig != nil {
		c := oidc.Config{
			Issuer:                    "https://oauth.geoaxis.gxaws.com",
			ClientID:                  request.GEOAxISConfig.ClientID,
			ClientSecret:              request.GEOAxISConfig.ClientSecret,
			RedirectURI:               "{{OIDCIdentityCallbackURL}}",
			GetUserInfo:               true,
			UserNameKey:               "email",
			InsecureSkipEmailVerified: false,
			InsecureEnableGroups:      true,
			Scopes: []string{
				"openid",
				"email",
				"groups",
				"uiasenterprise",
				"eiasenterprise",
			},
		}

		// un-redact
		if c.ClientSecret == RedactionMask {
			for _, idpConfig := range idpConfigs {
				if idpConfig.GEOAxISConfig != nil {
					c.ClientSecret = idpConfig.GEOAxISConfig.ClientSecret
				}
			}
		}

		c.ClaimMapping.GroupsKey = "group"

		// overrides and advanced options
		if request.GEOAxISConfig.Issuer != "" {
			c.Issuer = request.GEOAxISConfig.Issuer
		}
		if request.GEOAxISConfig.GetUserInfo != nil {
			c.GetUserInfo = *request.GEOAxISConfig.GetUserInfo
		}
		if request.GEOAxISConfig.UserNameKey != "" {
			c.UserNameKey = request.GEOAxISConfig.UserNameKey
		}
		if request.GEOAxISConfig.UserIDKey != "" {
			c.UserIDKey = request.GEOAxISConfig.UserIDKey
		}
		if request.GEOAxISConfig.PromptType != "" {
			c.PromptType = request.GEOAxISConfig.PromptType
		}
		if request.GEOAxISConfig.InsecureSkipEmailVerified != nil {
			c.InsecureSkipEmailVerified = *request.GEOAxISConfig.InsecureSkipEmailVerified
		}
		if request.GEOAxISConfig.InsecureEnableGroups != nil {
			c.InsecureEnableGroups = *request.GEOAxISConfig.InsecureEnableGroups
		}
		if len(request.GEOAxISConfig.Scopes) > 0 {
			c.Scopes = append(c.Scopes, request.GEOAxISConfig.Scopes...)
		}
		if len(request.GEOAxISConfig.HostedDomains) > 0 {
			c.HostedDomains = request.GEOAxISConfig.HostedDomains
		}
		if request.GEOAxISConfig.ClaimMapping.PreferredUsernameKey != "" {
			c.ClaimMapping.PreferredUsernameKey = request.GEOAxISConfig.ClaimMapping.PreferredUsernameKey
		}
		if request.GEOAxISConfig.ClaimMapping.EmailKey != "" {
			c.ClaimMapping.EmailKey = request.GEOAxISConfig.ClaimMapping.EmailKey
		}
		if request.GEOAxISConfig.ClaimMapping.GroupsKey != "" {
			c.ClaimMapping.GroupsKey = request.GEOAxISConfig.ClaimMapping.GroupsKey
		}
		connectorConfig = c

		dexConnectorInfo.Type = "oidc"
		dexConnectorInfo.ID = "geoaxis"
		dexConnectorInfo.Name = "GEOAxIS"
	} else {
		return nil, errors.New("provider not found")
	}

	// marshal connector config
	marshalledConnectorConfig, err := json.Marshal(connectorConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal connector config")
	}
	dexConnectorInfo.Config = marshalledConnectorConfig

	return &dexConnectorInfo, nil
}

func validateConfigureIdentityRequest(request ConfigureIdentityServiceRequest) error {
	missingFields := []string{}

	if request.AdminConsoleAddress == "" {
		missingFields = append(missingFields, "adminConsoleAddress")
	}

	if request.IdentityServiceAddress == "" {
		missingFields = append(missingFields, "identityServiceAddress")
	}

	if request.OIDCConfig != nil {
		if request.OIDCConfig.Issuer == "" {
			missingFields = append(missingFields, "issuer")
		}
		if request.OIDCConfig.ClientID == "" {
			missingFields = append(missingFields, "clientId")
		}
		if request.OIDCConfig.ClientSecret == "" {
			missingFields = append(missingFields, "clientSecret")
		}
	} else if request.GEOAxISConfig != nil {
		if request.GEOAxISConfig.ClientID == "" {
			missingFields = append(missingFields, "clientId")
		}
		if request.GEOAxISConfig.ClientSecret == "" {
			missingFields = append(missingFields, "clientSecret")
		}
	}

	var err error
	if len(missingFields) > 0 {
		errMsg := fmt.Sprintf("missing fields: %s", strings.Join(missingFields, ","))
		err = errors.New(errMsg)
	}

	return err
}

type GetIdentityServiceConfigResponse struct {
	Enabled                bool   `json:"enabled"`
	AdminConsoleAddress    string `json:"adminConsoleAddress"`
	IdentityServiceAddress string `json:"identityServiceAddress"`

	IDPConfig `json:",inline"`
}

func GetIdentityServiceConfig(w http.ResponseWriter, r *http.Request) {
	namespace := os.Getenv("POD_NAMESPACE")

	identityConfig, err := identity.GetConfig(r.Context(), namespace)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: return ingress config

	response := GetIdentityServiceConfigResponse{
		Enabled:                identityConfig.Spec.Enabled,
		AdminConsoleAddress:    identityConfig.Spec.AdminConsoleAddress,
		IdentityServiceAddress: identityConfig.Spec.IdentityServiceAddress,
	}

	idpConfigs, err := dexConnectorsToIDPConfigs(identityConfig.Spec.DexConnectors.Value)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, idpConfig := range idpConfigs {
		// redact
		if idpConfig.OIDCConfig != nil {
			idpConfig.OIDCConfig.ClientSecret = RedactionMask
		}
		if idpConfig.GEOAxISConfig != nil {
			idpConfig.GEOAxISConfig.ClientSecret = RedactionMask
		}

		response.IDPConfig = idpConfig
		// TODO: support for multiple connectors
		break
	}

	JSON(w, http.StatusOK, response)
}

func dexConnectorsToIDPConfigs(dexConnectors []kotsv1beta1.DexConnector) ([]IDPConfig, error) {
	conns, err := identity.IdentityDexConnectorsToDexTypeConnectors(dexConnectors)
	if err != nil {
		return nil, errors.Wrap(err, "failed to map identity dex connectors to dex type connectors")
	}

	idpConfigs := []IDPConfig{}
	for _, conn := range conns {
		switch c := conn.Config.(type) {
		case *oidc.Config:
			claimMapping := OIDCClaimMapping{
				PreferredUsernameKey: c.ClaimMapping.PreferredUsernameKey,
				EmailKey:             c.ClaimMapping.EmailKey,
				GroupsKey:            c.ClaimMapping.GroupsKey,
			}

			oidcConfig := &OIDCConfig{
				ConnectorID:               conn.ID,
				ConnectorName:             conn.Name,
				Issuer:                    c.Issuer,
				ClientID:                  c.ClientID,
				ClientSecret:              c.ClientSecret,
				GetUserInfo:               &c.GetUserInfo,
				UserNameKey:               c.UserNameKey,
				UserIDKey:                 c.UserIDKey,
				PromptType:                c.PromptType,
				InsecureSkipEmailVerified: &c.InsecureSkipEmailVerified,
				InsecureEnableGroups:      &c.InsecureEnableGroups,
				Scopes:                    c.Scopes,
				ClaimMapping:              claimMapping,
			}

			idpConfig := IDPConfig{}
			if conn.ID == "geoaxis" {
				idpConfig.GEOAxISConfig = oidcConfig
			} else {
				idpConfig.OIDCConfig = oidcConfig
			}
			idpConfigs = append(idpConfigs, idpConfig)
		}
	}
	return idpConfigs, nil
}
