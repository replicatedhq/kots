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
	AdminConsoleAddress    string         `json:"adminConsoleAddress"`
	IdentityServiceAddress string         `json:"identityServiceAddress"`
	OIDCConfig             *OIDCConfig    `json:"oidcConfig"`
	GEOAxISConfig          *GEOAxISConfig `json:"geoAxisConfig"`
}

type OIDCConfig struct {
	Issuer                    string   `json:"issuer"`
	ClientID                  string   `json:"clientId"`
	ClientSecret              string   `json:"clientSecret"`
	GetUserInfo               *bool    `json:"getUserInfo"`
	UserNameKey               string   `json:"userNameKey"`
	InsecureSkipEmailVerified *bool    `json:"insecureSkipEmailVerified"`
	InsecureEnableGroups      *bool    `json:"insecureEnableGroups"`
	Scopes                    []string `json:"scopes"`
}

type GEOAxISConfig struct {
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

	connectorInfo, err := getDexConnectorInfo(request)
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

	namespace := os.Getenv("POD_NAMESPACE")

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

func getDexConnectorInfo(request ConfigureIdentityServiceRequest) (*DexConnectorInfo, error) {
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

		dexConnectorInfo.Type = "oidc"
		dexConnectorInfo.ID = "openid"
		dexConnectorInfo.Name = "OpenID"
		connectorConfig = c
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

		c.ClaimMapping.GroupsKey = "group"

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

		dexConnectorInfo.Type = "oidc"
		dexConnectorInfo.ID = "geoaxis"
		dexConnectorInfo.Name = "GEOAxIS"
		connectorConfig = c
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
	IdentityProvider       string `json:"identityProvider"`
	Issuer                 string `json:"issuer"`
}

func GetIdentityServiceConfig(w http.ResponseWriter, r *http.Request) {
	namespace := os.Getenv("POD_NAMESPACE")

	identityConfig, err := identity.GetConfig(r.Context(), namespace)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: support types other than oidc
	// maybe redact the config and return it?

	// TODO: return ingress config

	response := GetIdentityServiceConfigResponse{
		Enabled:                identityConfig.Spec.Enabled,
		AdminConsoleAddress:    identityConfig.Spec.AdminConsoleAddress,
		IdentityServiceAddress: identityConfig.Spec.IdentityServiceAddress,
	}

	if len(identityConfig.Spec.DexConnectors.Value) > 0 {
		conn := identityConfig.Spec.DexConnectors.Value[0]
		response.IdentityProvider = conn.Name

		if len(conn.Config.Raw) != 0 {
			// unmarshal connector config
			var connectorConfig oidc.Config
			data := []byte(os.ExpandEnv(string(conn.Config.Raw)))
			err := json.Unmarshal(data, &connectorConfig)
			if err != nil {
				logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			response.Issuer = connectorConfig.Issuer
		}
	}

	JSON(w, http.StatusOK, response)
}
