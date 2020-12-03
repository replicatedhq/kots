package handlers

import (
	"encoding/json"
	"net/http"
	"os"

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

type ConfigureIdentityServiceRequest struct {
	AdminConsoleAddress    string `json:"adminConsoleAddress"`
	IdentityServiceAddress string `json:"identityServiceAddress"`

	IDPConfig `json:",inline"`
}

type OIDCConfig struct {
	ConnectorID               string   `json:"connectorId"`
	ConnectorName             string   `json:"connectorName"`
	Issuer                    string   `json:"issuer"`
	ClientID                  string   `json:"clientId"`
	ClientSecret              string   `json:"clientSecret"`
	GetUserInfo               *bool    `json:"getUserInfo"`
	UserNameKey               string   `json:"userNameKey"`
	InsecureSkipEmailVerified *bool    `json:"insecureSkipEmailVerified"`
	InsecureEnableGroups      *bool    `json:"insecureEnableGroups"`
	Scopes                    []string `json:"scopes"`
}

func ConfigureIdentityService(w http.ResponseWriter, r *http.Request) {
	request := ConfigureIdentityServiceRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.Error(err)
		response := ErrorResponse{Error: "failed to decode request body"}
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	namespace := os.Getenv("POD_NAMESPACE")

	previousConfig, err := identity.GetConfig(r.Context(), namespace)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get identity config"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	idpConfigs, err := dexConnectorsToIDPConfigs(previousConfig.Spec.DexConnectors.Value)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	connectorInfo, err := getDexConnectorInfo(request, idpConfigs)
	if err != nil {
		logger.Error(err)
		response := ErrorResponse{Error: "failed to get dex connector info"}
		JSON(w, http.StatusInternalServerError, response)
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
		logger.Error(err)
		response := ErrorResponse{Error: "failed to get ingress config"}
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if err := identity.ConfigValidate(identityConfig.Spec, ingressConfig.Spec); err != nil {
		logger.Error(err)
		response := ErrorResponse{Error: "failed to validate identity config"}
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if err := identity.SetConfig(r.Context(), namespace, identityConfig); err != nil {
		logger.Error(err)
		response := ErrorResponse{Error: "failed to set identity config"}
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	cfg, err := config.GetConfig()
	if err != nil {
		logger.Error(err)
		response := ErrorResponse{Error: "failed to get cluster config"}
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Error(err)
		response := ErrorResponse{Error: "failed to create kubernetes clientset"}
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	registryOptions, err := kotsadm.GetKotsadmOptionsFromCluster(namespace, clientset)
	if err != nil {
		logger.Error(err)
		response := ErrorResponse{Error: "failed to get kotsadm options from cluster"}
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if err := identity.Deploy(r.Context(), clientset, namespace, identityConfig, *ingressConfig, &registryOptions); err != nil {
		logger.Error(err)
		response := ErrorResponse{Error: "failed to deploy the identity service"}
		JSON(w, http.StatusInternalServerError, response)
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
		if c.ClientSecret == "" {
			for _, idpConfig := range idpConfigs {
				if idpConfig.OIDCConfig != nil {
					c.ClientSecret = idpConfig.OIDCConfig.ClientSecret
				}
			}
		}

		if request.OIDCConfig.GetUserInfo != nil {
			c.GetUserInfo = *request.OIDCConfig.GetUserInfo
		}
		if request.OIDCConfig.UserNameKey != "" {
			c.UserNameKey = request.OIDCConfig.UserNameKey
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
		if c.ClientSecret == "" {
			for _, idpConfig := range idpConfigs {
				if idpConfig.GEOAxISConfig != nil {
					c.ClientSecret = idpConfig.GEOAxISConfig.ClientSecret
				}
			}
		}

		c.ClaimMapping.GroupsKey = "group"

		if request.GEOAxISConfig.Issuer != "" {
			c.Issuer = request.GEOAxISConfig.Issuer
		}
		if request.GEOAxISConfig.GetUserInfo != nil {
			c.GetUserInfo = *request.GEOAxISConfig.GetUserInfo
		}
		if request.GEOAxISConfig.UserNameKey != "" {
			c.UserNameKey = request.GEOAxISConfig.UserNameKey
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

type GetIdentityServiceConfigResponse struct {
	Enabled                bool   `json:"enabled"`
	AdminConsoleAddress    string `json:"adminConsoleAddress"`
	IdentityServiceAddress string `json:"identityServiceAddress"`

	IDPConfig `json:",inline"`
}

type IDPConfig struct {
	OIDCConfig    *OIDCConfig `json:"oidcConfig"`
	GEOAxISConfig *OIDCConfig `json:"geoAxisConfig"`
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
			idpConfig.OIDCConfig.ClientSecret = ""
		}
		if idpConfig.GEOAxISConfig != nil {
			idpConfig.GEOAxISConfig.ClientSecret = ""
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
		return nil, errors.Wrap(err, "failed to unmarshal dex connectors")
	}

	idpConfigs := []IDPConfig{}
	for _, conn := range conns {
		switch c := conn.Config.(type) {
		case *oidc.Config:
			oidcConfig := &OIDCConfig{
				ConnectorID:               conn.ID,
				ConnectorName:             conn.Name,
				Issuer:                    c.Issuer,
				ClientID:                  c.ClientID,
				ClientSecret:              c.ClientSecret,
				GetUserInfo:               &c.GetUserInfo,
				UserNameKey:               c.UserNameKey,
				InsecureSkipEmailVerified: &c.InsecureSkipEmailVerified,
				InsecureEnableGroups:      &c.InsecureEnableGroups,
				Scopes:                    c.Scopes,
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
