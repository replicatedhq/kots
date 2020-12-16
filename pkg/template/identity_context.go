package template

import (
	"fmt"
	"strconv"
	"text/template"

	"github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	"github.com/replicatedhq/kots/pkg/ingress"
)

type identityCtx struct {
	identityConfig *v1beta1.IdentityConfig
	appInfo        *ApplicationInfo
	cipher         *crypto.AESCipher
}

func newIdentityCtx(identityConfig *v1beta1.IdentityConfig, appInfo *ApplicationInfo, cipher *crypto.AESCipher) identityCtx {
	return identityCtx{
		identityConfig: identityConfig,
		appInfo:        appInfo,
		cipher:         cipher,
	}
}

// FuncMap represents the available functions in the identityCtx.
func (ctx identityCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"IdentityServiceEnabled":      ctx.identityServiceEnabled,
		"IdentityServiceIssuerURL":    ctx.identityServiceIssuerURL,
		"IdentityServiceClientID":     ctx.identityServiceClientID,
		"IdentityServiceClientSecret": ctx.identityServiceClientSecret,
		"IdentityServiceRoles":        ctx.identityServiceRoles,
		"IdentityServiceName":         ctx.identityServiceName,
		"IdentityServicePort":         ctx.identityServicePort,
	}
}

func (ctx identityCtx) identityServiceEnabled() bool {
	if ctx.identityConfig == nil {
		return false
	}
	return ctx.identityConfig.Spec.Enabled
}

func (ctx identityCtx) identityServiceIssuerURL() string {
	if ctx.identityConfig == nil {
		return ""
	}
	if ctx.identityConfig.Spec.IdentityServiceAddress != "" {
		return ctx.identityConfig.Spec.IdentityServiceAddress
	}
	return fmt.Sprintf("%s/dex", ingress.GetAddress(ctx.identityConfig.Spec.IngressConfig))
}

func (ctx identityCtx) identityServiceClientID() string {
	if ctx.identityConfig == nil {
		return ""
	}
	return ctx.identityConfig.Spec.ClientID
}

func (ctx identityCtx) identityServiceClientSecret() (string, error) {
	if ctx.identityConfig == nil || ctx.cipher == nil {
		return "", nil
	}
	return ctx.identityConfig.Spec.ClientSecret.GetValue(*ctx.cipher)
}

func (ctx identityCtx) identityServiceRoles() map[string]interface{} {
	m := map[string]interface{}{}

	if ctx.identityConfig != nil {
		for _, g := range ctx.identityConfig.Spec.Groups {
			m[g.ID] = g.RoleIDs
		}
	}

	return m
}

func (ctx identityCtx) identityServiceName() string {
	if ctx.appInfo == nil {
		return ""
	}
	return identitytypes.ServiceName(ctx.appInfo.Slug)
}

func (ctx identityCtx) identityServicePort() string {
	if ctx.appInfo == nil {
		return ""
	}
	return strconv.Itoa(int(identitytypes.ServicePort()))
}
