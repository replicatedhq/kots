package handlers

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/dexidp/dex/connector/oidc"
	"github.com/gosimple/slug"
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
	IdentityProvider       string `json:"identityProvider"`
	Issuer                 string `json:"issuer"`
	ClientID               string `json:"clientId"`
	ClientSecret           string `json:"clientSecret"`
}

type ConfigureIdentityServiceResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func ConfigureIdentityService(w http.ResponseWriter, r *http.Request) {
	response := ConfigureIdentityServiceResponse{}

	request := ConfigureIdentityServiceRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.Error(err)
		response.Error = "failed to decode request body"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	namespace := os.Getenv("POD_NAMESPACE")

	ingressConfig, err := ingress.GetConfig(r.Context(), namespace)
	if err != nil {
		logger.Error(err)
		response.Error = "failed to get ingress config"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	// TODO: support types other than oidc
	connectorConfig := oidc.Config{
		Issuer:                    request.Issuer,
		ClientID:                  request.ClientID,
		ClientSecret:              request.ClientSecret,
		RedirectURI:               "{{OIDCIdentityCallbackURL}}",
		GetUserInfo:               true,
		UserNameKey:               "email",
		InsecureSkipEmailVerified: true,
		InsecureEnableGroups:      true,
		Scopes: []string{
			"openid",
			"email",
			"profile",
			"offline_access",
			"groups",
		},
	}

	providerID := slug.Make(request.IdentityProvider)
	if providerID == "geoaxis" {
		// issuer
		connectorConfig.Issuer = "https://oauth.geoaxis.gxaws.com"

		// scopes
		geoaxisAdditionalScopes := []string{"uiasenterprise", "eiasenterprise"}
		connectorConfig.Scopes = append(connectorConfig.Scopes, geoaxisAdditionalScopes...)

		// claim mapping
		connectorConfig.ClaimMapping.GroupsKey = "group"
	}

	// marshal connector config
	marshalledConnectorConfig, err := json.Marshal(connectorConfig)
	if err != nil {
		logger.Error(err)
		response.Error = "failed to marshal connector config"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

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
						Type: "oidc", // TODO: support other types
						Name: request.IdentityProvider,
						ID:   providerID,
						Config: runtime.RawExtension{
							Raw: marshalledConnectorConfig,
						},
					},
				},
			},
		},
	}

	if err := identity.ConfigValidate(identityConfig.Spec, ingressConfig.Spec); err != nil {
		logger.Error(err)
		response.Error = "failed to validate identity config"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if err := identity.SetConfig(r.Context(), namespace, identityConfig); err != nil {
		logger.Error(err)
		response.Error = "failed to set identity config"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	cfg, err := config.GetConfig()
	if err != nil {
		logger.Error(err)
		response.Error = "failed to get cluster config"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Error(err)
		response.Error = "failed to create kubernetes clientset"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	registryOptions, err := kotsadm.GetKotsadmOptionsFromCluster(namespace, clientset)
	if err != nil {
		logger.Error(err)
		response.Error = "failed to get kotsadm options from cluster"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if err := identity.Deploy(r.Context(), clientset, namespace, identityConfig, *ingressConfig, &registryOptions); err != nil {
		logger.Error(err)
		response.Error = "failed to deploy the identity service"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	response.Success = true

	JSON(w, http.StatusOK, response)
}
