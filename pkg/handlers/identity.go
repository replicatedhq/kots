package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	dextypes "github.com/replicatedhq/kots/pkg/dex/types"
	"github.com/replicatedhq/kots/pkg/handlers/types"
	"github.com/replicatedhq/kots/pkg/identity"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmidentity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/rbac"
	"github.com/replicatedhq/kots/pkg/render"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	RedactionMask = "--- REDACTED ---"
)

// TODO: separate request types for kotsadm and the app?
type ConfigureIdentityServiceRequest struct {
	AdminConsoleAddress     string                            `json:"adminConsoleAddress,omitempty"`
	IdentityServiceAddress  string                            `json:"identityServiceAddress,omitempty"`
	UseAdminConsoleSettings bool                              `json:"useAdminConsoleSettings,omitempty"`
	Groups                  []kotsv1beta1.IdentityConfigGroup `json:"groups,omitempty"`

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
	ClaimMapping              OIDCClaimMapping `json:"claimMapping,omitempty"`
}

type OIDCClaimMapping struct {
	PreferredUsernameKey string `json:"preferredUsername,omitempty"`
	EmailKey             string `json:"email,omitempty"`
	GroupsKey            string `json:"groups,omitempty"`
}

func (h *Handler) ConfigureIdentityService(w http.ResponseWriter, r *http.Request) {
	request := ConfigureIdentityServiceRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		err = errors.Wrap(err, "failed to decode request body")
		logger.Error(err)
		JSON(w, http.StatusBadRequest, types.NewErrorResponse(err))
		return
	}

	if err := validateConfigureIdentityRequest(request, false); err != nil {
		err = errors.Wrap(err, "failed to validate request")
		logger.Error(err)
		JSON(w, http.StatusBadRequest, types.NewErrorResponse(err))
		return
	}

	namespace := util.PodNamespace

	previousConfig, err := identity.GetConfig(r.Context(), namespace)
	if err != nil {
		err = errors.Wrap(err, "failed to get identity config")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// NOTE: we do not encrypt kotsadm config

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

	if len(request.Groups) > 0 {
		identityConfig.Spec.Groups = request.Groups
	}

	ingressConfig, err := ingress.GetConfig(r.Context(), namespace)
	if err != nil {
		err = errors.Wrap(err, "failed to get ingress config")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := identity.ValidateConfig(r.Context(), namespace, identityConfig, *ingressConfig); err != nil {
		err = errors.Wrap(err, "invalid identity config")
		logger.Error(err)
		JSON(w, http.StatusBadRequest, types.NewErrorResponse(err))
		return
	}

	// TODO: validate dex issuer
	if err := identity.ValidateConnection(r.Context(), namespace, identityConfig, *ingressConfig); err != nil {
		if _, ok := errors.Cause(err).(*identity.ErrorConnection); ok {
			err = errors.Wrap(err, "invalid connection")
			logger.Error(err)
			JSON(w, http.StatusBadRequest, types.NewErrorResponse(err))
			return
		}
		err = errors.Wrap(err, "failed to validate identity connection")
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

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		err = errors.Wrap(err, "failed to get k8s client set")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	registryConfig, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
	if err != nil {
		err = errors.Wrap(err, "failed to get kotsadm options from cluster")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	apps, err := store.GetStore().ListInstalledAppSlugs()
	if err != nil {
		err = errors.Wrap(err, "failed to list installed apps")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	applyAppBranding := len(apps) == 1

	proxyEnv := map[string]string{
		"HTTP_PROXY":  os.Getenv("HTTP_PROXY"),
		"HTTPS_PROXY": os.Getenv("HTTPS_PROXY"),
		"NO_PROXY":    os.Getenv("NO_PROXY"),
	}
	if err := identity.Deploy(r.Context(), clientset, namespace, identityConfig, *ingressConfig, &registryConfig, proxyEnv, applyAppBranding); err != nil {
		err = errors.Wrap(err, "failed to deploy the identity service")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := types.ErrorResponse{
		Success: true,
	}

	JSON(w, http.StatusOK, response)
}

func (h *Handler) ConfigureAppIdentityService(w http.ResponseWriter, r *http.Request) {
	request := ConfigureIdentityServiceRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		err = errors.Wrap(err, "failed to decode request body")
		logger.Error(err)
		JSON(w, http.StatusBadRequest, types.NewErrorResponse(err))
		return
	}

	if err := validateConfigureIdentityRequest(request, true); err != nil {
		err = errors.Wrap(err, "failed to validate request")
		logger.Error(err)
		JSON(w, http.StatusBadRequest, types.NewErrorResponse(err))
		return
	}

	a, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	latestSequence, err := store.GetStore().GetLatestAppSequence(a.ID, true)
	if err != nil {
		err = errors.Wrap(err, "failed to get latest app sequence")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		err = errors.Wrap(err, "failed to create temp dir")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(a.ID, latestSequence, archiveDir)
	if err != nil {
		err = errors.Wrap(err, "failed to get current app version archive")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
	if err != nil {
		err = errors.Wrap(err, "failed to load kots kinds from path")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if kotsKinds.Identity == nil {
		err := errors.New("identity spec not found")
		logger.Error(err)
		JSON(w, http.StatusBadRequest, types.NewErrorResponse(err))
		return
	}

	identityConfigFile := filepath.Join(archiveDir, "upstream", "userdata", "identityconfig.yaml")
	if _, err := os.Stat(identityConfigFile); os.IsNotExist(err) {
		f, err := kotsadmidentity.InitAppIdentityConfig(a.Slug)
		if err != nil {
			err = errors.Wrap(err, "failed to init identity config")
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(f)
		identityConfigFile = f
	} else if err != nil {
		err = errors.Wrap(err, "failed to stat identity config file")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := os.ReadFile(identityConfigFile)
	if err != nil {
		err = errors.Wrap(err, "failed to read identityconfig file")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s, err := kotsutil.LoadIdentityConfigFromContents(b)
	if err != nil {
		err = errors.Wrap(err, "failed to decode identity service config")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	identityConfig := *s

	dexConnectors, err := identityConfig.Spec.DexConnectors.GetValue()
	if err != nil {
		err = errors.Wrap(err, "failed to decrypt dex connectors")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	idpConfigs, err := dexConnectorsToIDPConfigs(dexConnectors)
	if err != nil {
		err = errors.Wrap(err, "failed to get idp configs from dex connectors")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if request.UseAdminConsoleSettings {
		// existing idpConfigs are used for secret un-redaction
		// if secrets in the request are not redacted, this won't have any effect
		// the user has the option to use kotsadm identity config
		// in that case, secrets should be retrieved from kotsadm identity config not the app identity config
		namespace := util.PodNamespace

		kotsadmIdentityConfig, err := identity.GetConfig(r.Context(), namespace)
		if err != nil {
			err = errors.Wrap(err, "failed to get kots identity config")
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// NOTE: we do not encrypt kotsadm config

		idpConfigs, err = dexConnectorsToIDPConfigs(kotsadmIdentityConfig.Spec.DexConnectors.Value)
		if err != nil {
			err = errors.Wrap(err, "failed to get kotsadm idp configs from dex connectors")
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	connectorInfo, err := getDexConnectorInfo(request, idpConfigs)
	if err != nil {
		err = errors.Wrap(err, "failed to get dex connector info")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: handle ingress config
	identityConfig.Spec.Enabled = true
	identityConfig.Spec.DisablePasswordAuth = true
	identityConfig.Spec.Groups = request.Groups
	identityConfig.Spec.DexConnectors = kotsv1beta1.DexConnectors{
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
	}

	namespace := util.PodNamespace

	// TODO: handle configuring ingress for the app?
	// TODO: validate dex issuer
	ingressConfig := kotsv1beta1.IngressConfig{}
	if err := identity.ValidateConnection(r.Context(), namespace, identityConfig, ingressConfig); err != nil {
		if _, ok := errors.Cause(err).(*identity.ErrorConnection); ok {
			err = errors.Wrap(err, "invalid connection")
			logger.Error(err)
			JSON(w, http.StatusBadRequest, types.NewErrorResponse(err))
			return
		}
		err = errors.Wrap(err, "failed to validate identity connection")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	kotsKinds.IdentityConfig = &identityConfig

	identityConfigSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "IdentityConfig")
	if err != nil {
		err = errors.Wrap(err, "failed to marshal config values spec")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "identityconfig.yaml"), []byte(identityConfigSpec), 0644); err != nil {
		err = errors.Wrap(err, "failed to write identityconfig.yaml to upstream/userdata")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to list downstreams for app")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to get registry settings")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	nextAppSequence, err := store.GetStore().GetNextAppSequence(a.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to get next app sequence")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = render.RenderDir(rendertypes.RenderDirOptions{
		ArchiveDir:       archiveDir,
		App:              a,
		Downstreams:      downstreams,
		RegistrySettings: registrySettings,
		Sequence:         nextAppSequence,
		ReportingInfo:    reporting.GetReportingInfo(a.ID),
	})
	if err != nil {
		err = errors.Wrap(err, "failed to render archive directory")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newSequence, err := store.GetStore().CreateAppVersion(a.ID, &latestSequence, archiveDir, "Identity Service", false, false, false)
	if err != nil {
		err = errors.Wrap(err, "failed to create an app version")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := preflight.Run(a.ID, a.Slug, newSequence, a.IsAirgap, false, archiveDir); err != nil {
		err = errors.Wrap(err, "failed to run preflights")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := types.ErrorResponse{
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

func validateConfigureIdentityRequest(request ConfigureIdentityServiceRequest, isAppConfig bool) error {
	missingFields := []string{}

	if !isAppConfig && request.AdminConsoleAddress == "" {
		missingFields = append(missingFields, "adminConsoleAddress")
	}

	if !isAppConfig && request.IdentityServiceAddress == "" {
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
	Enabled                bool                              `json:"enabled"`
	AdminConsoleAddress    string                            `json:"adminConsoleAddress,omitempty"`
	IdentityServiceAddress string                            `json:"identityServiceAddress,omitempty"`
	Groups                 []kotsv1beta1.IdentityConfigGroup `json:"groups,omitempty"`
	Roles                  []kotsv1beta1.IdentityRole        `json:"roles,omitempty"`

	IDPConfig `json:",inline"`
}

func (h *Handler) GetIdentityServiceConfig(w http.ResponseWriter, r *http.Request) {
	namespace := util.PodNamespace

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
		Groups:                 identityConfig.Spec.Groups,
	}

	roles := []kotsv1beta1.IdentityRole{}
	defaultRoles := rbac.DefaultRoles()
	for _, defaultRole := range defaultRoles {
		role := kotsv1beta1.IdentityRole{
			ID:          defaultRole.ID,
			Name:        defaultRole.Name,
			Description: defaultRole.Description,
		}
		roles = append(roles, role)
	}
	response.Roles = roles

	// NOTE: we do not encrypt kotsadm config

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

func (h *Handler) GetAppIdentityServiceConfig(w http.ResponseWriter, r *http.Request) {
	a, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	latestSequence, err := store.GetStore().GetLatestAppSequence(a.ID, true)
	if err != nil {
		err = errors.Wrap(err, "failed to get latest app sequence")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		err = errors.Wrap(err, "failed to create temp dir")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(a.ID, latestSequence, archiveDir)
	if err != nil {
		err = errors.Wrap(err, "failed to get current app version archive")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
	if err != nil {
		err = errors.Wrap(err, "failed to load kotskinds from path")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if kotsKinds.Identity == nil {
		err := errors.New("identity spec not found")
		logger.Error(err)
		JSON(w, http.StatusBadRequest, types.NewErrorResponse(err))
		return
	}

	response := GetIdentityServiceConfigResponse{
		Roles: kotsKinds.Identity.Spec.Roles,
	}

	if kotsKinds.IdentityConfig == nil {
		// identity service not configured yet
		JSON(w, http.StatusOK, response)
		return
	}

	// TODO: return ingress config
	response.Enabled = kotsKinds.IdentityConfig.Spec.Enabled
	response.Groups = kotsKinds.IdentityConfig.Spec.Groups

	dexConnectors, err := kotsKinds.IdentityConfig.Spec.DexConnectors.GetValue()
	if err != nil {
		err = errors.Wrap(err, "failed to decrypt dex connectors")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(dexConnectors) == 0 {
		// no connectors, return default values
		response.IDPConfig = getDefaultIDPConfig()
		JSON(w, http.StatusOK, response)
		return
	}

	idpConfigs, err := dexConnectorsToIDPConfigs(dexConnectors)
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
	conns, err := identitydeploy.DexConnectorsToDexTypeConnectors(dexConnectors)
	if err != nil {
		return nil, errors.Wrap(err, "failed to map identity dex connectors to dex type connectors")
	}

	idpConfigs := []IDPConfig{}
	for _, conn := range conns {
		switch c := conn.Config.(type) {
		case *dextypes.OIDCConfig:
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

func getDefaultOIDCConfig(isGeoAxis bool) *dextypes.OIDCConfig {
	c := dextypes.OIDCConfig{
		GetUserInfo:               true,
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
		c.UserNameKey = "email"
		c.Scopes = append(c.Scopes, "uiasenterprise")
		c.Scopes = append(c.Scopes, "eiasenterprise")
		c.ClaimMapping.GroupsKey = "group"
		c.InsecureSkipEmailVerified = true
	}

	return &c
}

func identityOIDCToOIDCConfig(identityOIDC *OIDCConfig, idpConfigs []IDPConfig, isGeoAxis bool) *dextypes.OIDCConfig {
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
			} else if idpConfig.GEOAxISConfig != nil {
				c.ClientSecret = idpConfig.GEOAxISConfig.ClientSecret
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

func oidcConfigToIdentityOIDC(c *dextypes.OIDCConfig, conn *dextypes.Connector) *OIDCConfig {
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
