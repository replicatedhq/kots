package template

import (
	"fmt"
	"text/template"

	"github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/ingress"
)

type identityCtx struct {
	identityConfig *v1beta1.IdentityConfig
}

func newIdentityCtx(identityConfig *v1beta1.IdentityConfig) identityCtx {
	return identityCtx{
		identityConfig: identityConfig,
	}
}

// FuncMap represents the available functions in the identityCtx.
func (ctx identityCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"IdentityServiceEnabled":          ctx.identityServiceEnabled,
		"IdentityServiceIssuerURL":        ctx.identityServiceIssuerURL,
		"IdentityServiceClientID":         ctx.identityServiceClientID,
		"IdentityServiceClientSecret":     ctx.identityServiceClientSecret,
		"IdentityServiceRestrictedGroups": ctx.identityServiceRestrictedGroups,
		"IdentityServiceRoles":            ctx.identityServiceRoles,
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

func (ctx identityCtx) identityServiceClientSecret() string {
	if ctx.identityConfig == nil {
		return ""
	}
	return ctx.identityConfig.Spec.ClientSecret
}

func (ctx identityCtx) identityServiceRestrictedGroups() []string {
	if ctx.identityConfig == nil {
		return []string{}
	}

	groups := []string{}
	for _, g := range ctx.identityConfig.Spec.Groups {
		groups = append(groups, g.ID)
	}

	return groups
}

func (ctx identityCtx) identityServiceRoles() map[string][]string {
	if ctx.identityConfig == nil {
		return map[string][]string{}
	}
	return map[string][]string{} // TODO (salah)
}
