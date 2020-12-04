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
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
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

	if err := identity.ConfigValidate(r.Context(), namespace, identityConfig, *ingressConfig, true); err != nil {
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
		c := identityOIDCToOIDCConfig(request.OIDCConfig, idpConfigs, false)
		connectorConfig = *c

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
		c := identityOIDCToOIDCConfig(request.GEOAxISConfig, idpConfigs, true)
		connectorConfig = *c

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
		if request.OIDCConfig.ConnectorName == "" {
			missingFields = append(missingFields, "connectorName")
		}
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

	if len(identityConfig.Spec.DexConnectors.Value) == 0 {
		// no connectors, return default values
		response.IDPConfig = getDefaultIDPConfig()
		JSON(w, http.StatusOK, response)
		return
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
			oidcConfig := oidcConfigToIdentityOIDC(c, &conn)
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

func getDefaultIDPConfig() IDPConfig {
	idpConfig := IDPConfig{}
	idpConfig.OIDCConfig = oidcConfigToIdentityOIDC(getDefaultOIDCConfig(false), nil)
	idpConfig.GEOAxISConfig = oidcConfigToIdentityOIDC(getDefaultOIDCConfig(true), nil)
	return idpConfig
}

func getDefaultOIDCConfig(isGeoAxis bool) *oidc.Config {
	c := oidc.Config{
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

	if isGeoAxis {
		c.Issuer = "https://oauth.geoaxis.gxaws.com"
		c.Scopes = append(c.Scopes, "uiasenterprise")
		c.Scopes = append(c.Scopes, "eiasenterprise")
		c.ClaimMapping.GroupsKey = "group"
	}

	return &c
}

func identityOIDCToOIDCConfig(identityOIDC *OIDCConfig, idpConfigs []IDPConfig, isGeoAxis bool) *oidc.Config {
	c := getDefaultOIDCConfig(isGeoAxis)

	if identityOIDC.Issuer != "" {
		c.Issuer = identityOIDC.Issuer
	}
	c.ClientID = identityOIDC.ClientID
	c.ClientSecret = identityOIDC.ClientSecret

	// un-redact
	if c.ClientSecret == RedactionMask {
		for _, idpConfig := range idpConfigs {
			if idpConfig.OIDCConfig != nil {
				c.ClientSecret = idpConfig.OIDCConfig.ClientSecret
			}
		}
	}

	// overrides and advanced options
	if identityOIDC.GetUserInfo != nil {
		c.GetUserInfo = *identityOIDC.GetUserInfo
	}
	if identityOIDC.UserNameKey != "" {
		c.UserNameKey = identityOIDC.UserNameKey
	}
	if identityOIDC.UserIDKey != "" {
		c.UserIDKey = identityOIDC.UserIDKey
	}
	if identityOIDC.PromptType != "" {
		c.PromptType = identityOIDC.PromptType
	}
	if identityOIDC.InsecureSkipEmailVerified != nil {
		c.InsecureSkipEmailVerified = *identityOIDC.InsecureSkipEmailVerified
	}
	if identityOIDC.InsecureEnableGroups != nil {
		c.InsecureEnableGroups = *identityOIDC.InsecureEnableGroups
	}
	if len(identityOIDC.Scopes) > 0 {
		c.Scopes = identityOIDC.Scopes
	}
	if len(identityOIDC.HostedDomains) > 0 {
		c.HostedDomains = identityOIDC.HostedDomains
	}
	if identityOIDC.ClaimMapping.PreferredUsernameKey != "" {
		c.ClaimMapping.PreferredUsernameKey = identityOIDC.ClaimMapping.PreferredUsernameKey
	}
	if identityOIDC.ClaimMapping.EmailKey != "" {
		c.ClaimMapping.EmailKey = identityOIDC.ClaimMapping.EmailKey
	}
	if identityOIDC.ClaimMapping.GroupsKey != "" {
		c.ClaimMapping.GroupsKey = identityOIDC.ClaimMapping.GroupsKey
	}

	return c
}

func oidcConfigToIdentityOIDC(c *oidc.Config, conn *dextypes.Connector) *OIDCConfig {
	claimMapping := OIDCClaimMapping{
		PreferredUsernameKey: c.ClaimMapping.PreferredUsernameKey,
		EmailKey:             c.ClaimMapping.EmailKey,
		GroupsKey:            c.ClaimMapping.GroupsKey,
	}

	oidcConfig := OIDCConfig{
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

	if conn != nil {
		oidcConfig.ConnectorID = conn.ID
		oidcConfig.ConnectorName = conn.Name
	}

	return &oidcConfig
}
