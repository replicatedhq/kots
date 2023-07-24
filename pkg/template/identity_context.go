package template

import (
	"strconv"
	"text/template"

	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	"github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

type identityCtx struct {
	identityConfig *v1beta1.IdentityConfig
	appInfo        *ApplicationInfo
}

func newIdentityCtx(identityConfig *v1beta1.IdentityConfig, appInfo *ApplicationInfo) identityCtx {
	return identityCtx{
		identityConfig: identityConfig,
		appInfo:        appInfo,
	}
}

// FuncMap represents the available functions in the identityCtx.
func (ctx identityCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"IdentityServiceEnabled":      ctx.identityServiceEnabled,
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

func (ctx identityCtx) identityServiceClientID() string {
	if ctx.identityConfig == nil {
		return ""
	}
	return ctx.identityConfig.Spec.ClientID
}

func (ctx identityCtx) identityServiceClientSecret() (string, error) {
	if ctx.identityConfig == nil {
		return "", nil
	}
	return ctx.identityConfig.Spec.ClientSecret.GetValue()
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
